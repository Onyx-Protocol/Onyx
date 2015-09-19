package appdb

import (
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
	"chain/strings"
)

// CreateActivityItems writes new activity items given a tx.
// Must be called inside a db transaction, but after the utxos are
// updated for this tx.
func CreateActivityItems(ctx context.Context, tx *wire.MsgTx, txTime time.Time) error {
	defer metrics.RecordElapsed(time.Now())
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	wids, addrIsChange, err := getWalletsAndChangeInTx(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "getting wallets in tx")
	}

	for _, w := range wids {
		err = walletActivityItem(ctx, tx, w, addrIsChange, txTime)
		if err != nil {
			msg := "writing item for wallet " + w
			return errors.Wrap(err, msg)
		}
	}

	return nil
}

// getWalletsAndChangeInTx takes a transaction and returns two things:
// a unique list of all the wallet ids associated with that transaction, and
// a map, address ids as keys, and values specifying if they're change addresses or not.
func getWalletsAndChangeInTx(ctx context.Context, tx *wire.MsgTx) ([]string, map[string]bool, error) {
	const q = `SELECT wallet_id, address_id FROM utxos WHERE txid=$1`

	var txids []string
	for _, i := range tx.TxIn {
		txids = append(txids, i.PreviousOutPoint.Hash.String())
	}
	txids = append(txids, tx.TxSha().String())

	var addrs []string
	var wallets []string
	for _, tid := range txids {
		rows, err := pg.FromContext(ctx).Query(q, tid)
		if err != nil {
			return nil, nil, errors.Wrap(err, "query")
		}
		defer rows.Close()

		for rows.Next() {
			var w, addr string
			err = rows.Scan(&w, &addr)
			wallets = append(wallets, w)
			addrs = append(addrs, addr)
		}
		if rows.Err() != nil {
			return nil, nil, errors.Wrap(rows.Err(), "rows")
		}
		rows.Close()
	}

	sort.Strings(wallets)
	wallets = strings.Uniq(wallets)

	const changeQ = `SELECT is_change FROM addresses WHERE id=$1`

	addrIsChange := make(map[string]bool)
	for _, a := range addrs {
		var isChange bool
		err := pg.FromContext(ctx).QueryRow(changeQ, a).Scan(&isChange)
		if err != nil {
			return nil, nil, errors.Wrap(err, "change row scan")
		}

		if isChange {
			addrIsChange[a] = true
		}
	}

	return wallets, addrIsChange, nil
}

func walletActivityItem(ctx context.Context, tx *wire.MsgTx, walletID string, addrIsChange map[string]bool, txTime time.Time) error {
	var (
		insByBucket  = make(map[string]map[string]int64)
		insByAddr    = make(map[string]map[string]int64)
		outsByBucket = make(map[string]map[string]int64)
		outsByAddr   = make(map[string]map[string]int64)
		buckets      []string
	)

	var (
		wid   string
		bid   string
		addr  string
		asset string
		amt   int64
	)

	const inputQ = `
		SELECT wallet_id, bucket_id, address_id, asset_id, amount
		FROM utxos
		WHERE txid=$1
	`

	// Pool all inputs by bucket or address.
	for _, in := range tx.TxIn {
		rows, err := pg.FromContext(ctx).Query(inputQ, in.PreviousOutPoint.Hash.String())
		if err != nil {
			return errors.Wrap(err, "querying inputs")
		}
		defer rows.Close()

		for rows.Next() {
			err = rows.Scan(&wid, &bid, &addr, &asset, &amt)
			if err != nil {
				return errors.Wrap(err, "row scan")
			}

			// We only want to track which bucket this came from
			// if this input is part of this wallet. Otherwise,
			// we only track the address.
			if wid == walletID {
				if insByBucket[bid] == nil {
					insByBucket[bid] = make(map[string]int64)
					buckets = append(buckets, bid)
				}

				insByBucket[bid][asset] += amt
			} else {
				if insByAddr[addr] == nil {
					insByAddr[addr] = make(map[string]int64)
				}

				insByAddr[addr][asset] += amt
			}
		}

		if rows.Err() != nil {
			return errors.Wrap(rows.Err(), "input tx rows")
		}

		rows.Close()
	}

	const outputQ = `
		SELECT wallet_id, bucket_id, address_id, asset_id, amount
		FROM utxos
		WHERE txid=$1
	`

	// Pool all outputs by bucket or address.
	hash := tx.TxSha().String()
	rows, err := pg.FromContext(ctx).Query(outputQ, hash)
	if err != nil {
		return errors.Wrap(err, "query output utxos")
	}
	defer rows.Close()

	// Iterate through all outputs:
	for rows.Next() {
		err = rows.Scan(&wid, &bid, &addr, &asset, &amt)
		if err != nil {
			return errors.Wrap(err, "row scan outputs")
		}

		// We only want to track which bucket
		// this came from if this output is part of this wallet.
		// Otherwise, we only track the address.
		if wid == walletID {

			// If this output is part of this wallet, it might
			// be a change address.
			// We only show the net effect on a bucket,
			// so if this address is a change address, we
			// do two things:
			//
			// 1) Don't add it to the list of outputs.
			// 2) Subtract it from inputs.
			if addrIsChange[addr] {
				if insByBucket[bid] != nil {
					insByBucket[bid][asset] -= amt
				} else {
					// If this addr is a change addr,
					// there should be an input that could
					// create said change.
					return errors.New("change addr without corresponding input")
				}
			} else {
				if outsByBucket[bid] == nil {
					outsByBucket[bid] = make(map[string]int64)
					buckets = append(buckets, bid)
				}

				outsByBucket[bid][asset] += amt
			}
		} else {
			if outsByAddr[addr] == nil {
				outsByAddr[addr] = make(map[string]int64)
			}

			outsByAddr[addr][asset] += amt
		}
	}

	sort.Strings(buckets)
	buckets = strings.Uniq(buckets)

	// Create Activity Item data blob. The Activity Item
	// blobs are json objects that look something like this:
	//
	// {
	//   "txid": txid,
	//   "inputs": [
	//     {
	//       "asset_id": aid,
	//       "amount": 10,
	//       "bucket_id": bid
	//     },
	//     {
	//       "asset_id": aid2,
	//       "amount": 10,
	//       "address": addr1
	//     }
	//   ],
	//   "outputs": [ same format as inputs ]
	// }
	//
	// These data blobs are stored in a json field on the activity table,
	// along with the wallet_id and tx_id.

	var (
		inputsBlob  []map[string]interface{}
		outputsBlob []map[string]interface{}
	)

	// TODO(tess): Make sure that insByAddr, insByBucket, outsByAddr, and
	// outsByBucket are traversed in order. htnote solves this problem
	// like this: https://github.com/chain-engineering/htnote/blob/master/payload.go#L183
	for addr, addrMap := range insByAddr {
		for asset, amt := range addrMap {
			i := make(map[string]interface{})
			i["asset_id"] = asset
			i["amount"] = amt
			i["address"] = addr

			inputsBlob = append(inputsBlob, i)
		}
	}
	for bckt, bcktMap := range insByBucket {
		for asset, amt := range bcktMap {
			i := make(map[string]interface{})
			i["asset_id"] = asset
			i["amount"] = amt
			i["bucket_id"] = bckt

			inputsBlob = append(inputsBlob, i)
		}
	}
	for addr, addrMap := range outsByAddr {
		for asset, amt := range addrMap {
			o := make(map[string]interface{})
			o["asset_id"] = asset
			o["amount"] = amt
			o["address"] = addr

			outputsBlob = append(outputsBlob, o)
		}
	}
	for bckt, bcktMap := range outsByBucket {
		for asset, amt := range bcktMap {
			o := make(map[string]interface{})
			o["asset_id"] = asset
			o["amount"] = amt
			o["bucket_id"] = bckt

			outputsBlob = append(outputsBlob, o)
		}
	}

	data := make(map[string]interface{})
	data["inputs"] = inputsBlob
	data["outputs"] = outputsBlob
	data["txid"] = hash
	data["transaction_time"] = txTime.UTC()

	// Now insert the data blob, along with the other tx information.
	const insertQ = `
		INSERT INTO activity (wallet_id, data, txid)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	blob, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "json marshalling data")
	}

	var actID string
	err = pg.FromContext(ctx).QueryRow(insertQ, walletID, blob, hash).Scan(&actID)
	if err != nil {
		return errors.Wrap(err, "inserting activity item")
	}

	const insertBucketQ = `
		INSERT INTO activity_buckets (activity_id, bucket_id) VALUES ($1, $2)
	`

	for _, bid := range buckets {
		_, err := pg.FromContext(ctx).Exec(insertBucketQ, actID, bid)
		if err != nil {
			return errors.Wrap(err, "inserting activity bucket item")
		}
	}

	return nil
}

func WalletActivity(ctx context.Context, walletID string, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT id, data FROM activity
		WHERE wallet_id=$1 AND (($2 = '') OR (id < $2))
		ORDER BY id DESC LIMIT $3
	`

	rows, err := pg.FromContext(ctx).Query(q, walletID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	return activityItemsFromRows(rows)
}

func BucketActivity(ctx context.Context, bucketID string, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT a.id, a.data
		FROM activity AS a
		LEFT JOIN activity_buckets AS ab
		ON a.id=ab.activity_id
		WHERE ab.bucket_id=$1 AND (($2 = '') OR (a.id < $2))
		ORDER BY a.id DESC LIMIT $3
	`

	rows, err := pg.FromContext(ctx).Query(q, bucketID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	return activityItemsFromRows(rows)
}

func activityItemsFromRows(rows *sql.Rows) (items []*json.RawMessage, last string, err error) {
	for rows.Next() {
		var a []byte
		err := rows.Scan(&last, &a)
		if err != nil {
			err = errors.Wrap(err, "row scan")
			return nil, "", err
		}

		items = append(items, (*json.RawMessage)(&a))
	}

	if rows.Err() != nil {
		err = errors.Wrap(rows.Err(), "rows")
		return nil, "", err
	}

	return items, last, nil
}

func WalletTxActivity(ctx context.Context, walletID, txID string) (*json.RawMessage, error) {
	q := `
		SELECT data FROM activity
		WHERE wallet_id=$1 AND txid=$2
	`

	var a []byte
	err := pg.FromContext(ctx).QueryRow(q, walletID, txID).Scan(&a)
	return (*json.RawMessage)(&a), err
}
