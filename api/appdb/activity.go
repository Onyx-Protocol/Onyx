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

// Errors return by functions in this file
var (
	ErrInvalidIssuanceActivity = errors.New("cannot generate activity for invalid issuance")
)

// WriteActivity generates formatted activity history for the given transaction.
// This must be called after the output UTXOs have been generated, but before
// the input UTXOs have been deleted. The supplied context must contain a
// database transaction.
func WriteActivity(ctx context.Context, tx *wire.MsgTx, txTime time.Time) error {
	defer metrics.RecordElapsed(time.Now())
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction

	txHash := tx.TxSha().String()

	// Get detailed UTXO information for the transaction's inputs and outputs.
	var (
		hashes  []string
		indexes []uint32
	)
	for _, i := range tx.TxIn {
		hashes = append(hashes, i.PreviousOutPoint.Hash.String())
		indexes = append(indexes, i.PreviousOutPoint.Index)
	}
	ins, err := getActUTXOs(ctx, hashes, indexes)
	if err != nil {
		return errors.Wrap(err, "get input utxos")
	}

	outs, err := getActUTXOsByTx(ctx, txHash)
	if err != nil {
		return errors.Wrap(err, "get output utxos")
	}

	// Extract IDs for all resources involved in the transaction. The lists
	// should not contain duplicates.
	allUTXOs := append(append([]*actUTXO{}, ins...), outs...)
	assetIDs, addrIDs, bucketIDs, walletIDs, walletBuckets := getIDsFromUTXOs(allUTXOs)

	// Gather additional data on relevant addresses.
	actAddrs, err := getActAddrs(ctx, addrIDs)
	if err != nil {
		return errors.Wrap(err, "get addresses")
	}

	var (
		addresses   = make(map[string]string)
		changeAddrs []string
	)
	for _, a := range actAddrs {
		addresses[a.id] = a.address
		if a.isChange {
			changeAddrs = append(changeAddrs, a.id)
		}
	}

	// Gather additional data on relevant buckets.
	actBuckets, err := getActBuckets(ctx, bucketIDs)
	if err != nil {
		return errors.Wrap(err, "get buckets")
	}

	bucketLabels := make(map[string]string)
	for _, b := range actBuckets {
		bucketLabels[b.id] = b.label
	}

	// Gather additional data on relevant assets.
	actAssets, err := getActAssets(ctx, assetIDs)
	if err != nil {
		return errors.Wrap(err, "get assets")
	}

	assetLabels := make(map[string]string)
	for _, a := range actAssets {
		assetLabels[a.id] = a.label
	}

	// Wallet activity
	for _, walletID := range walletIDs {
		r := coalesceActivity(ins, outs, walletBuckets[walletID], changeAddrs)
		inAct, outAct := createActEntries(r, addresses, assetLabels, bucketLabels)

		data, err := serializeActvity(txHash, txTime, inAct, outAct)
		if err != nil {
			return errors.Wrap(err, "serialize activity")
		}

		err = writeWalletActivity(ctx, walletID, txHash, data, walletBuckets[walletID])
		if err != nil {
			return errors.Wrap(err, "writing activity for wallet", walletID)
		}
	}

	// Issuance activity
	if isIssuance(tx) {
		// Only one asset may be issued per transaction.
		if len(actAssets) != 1 {
			return errors.Wrap(ErrInvalidIssuanceActivity, "asset count:", len(actAssets))
		}

		var visibleBuckets []string
		for _, b := range actBuckets {
			if b.appID == actAssets[0].appID {
				visibleBuckets = append(visibleBuckets, b.id)
			}
		}

		r := coalesceActivity(ins, outs, visibleBuckets, changeAddrs)
		inAct, outAct := createActEntries(r, addresses, assetLabels, bucketLabels)

		data, err := serializeActvity(txHash, txTime, inAct, outAct)
		if err != nil {
			return errors.Wrap(err, "serialize activity")
		}

		err = writeIssuanceActivity(ctx, actAssets[0], txHash, data)
		if err != nil {
			return errors.Wrap(err, "writing activity for asset group", actAssets[0].agID)
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

func AssetGroupActivity(ctx context.Context, agID string, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT id, data FROM issuance_activity
		WHERE asset_group_id = $1 AND (($2 = '') OR (id < $2))
		ORDER BY id DESC LIMIT $3
	`
	rows, err := pg.FromContext(ctx).Query(q, agID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	return activityItemsFromRows(rows)
}

func AssetActivity(ctx context.Context, assetID string, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT ia.id, ia.data
		FROM issuance_activity AS ia
		LEFT JOIN issuance_activity_assets AS j
		ON ia.id = j.issuance_activity_id
		WHERE j.asset_id = $1 AND (($2 = '') OR (ia.id < $2))
		ORDER BY ia.id DESC LIMIT $3
	`
	rows, err := pg.FromContext(ctx).Query(q, assetID, prev, limit)
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
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "transaction id: %v", txID)
	}
	return (*json.RawMessage)(&a), err
}

type actUTXO struct {
	assetID string
	amount  int64

	addrID   string
	bucketID string
	walletID string
}

type actAddr struct {
	id       string
	address  string
	isChange bool
}

type actAsset struct {
	id    string
	label string
	agID  string
	appID string
}

type actBucket struct {
	id       string
	label    string
	walletID string
	appID    string
}

type txRawActivity struct {
	insByA  map[string]map[string]int64
	insByB  map[string]map[string]int64
	outsByA map[string]map[string]int64
	outsByB map[string]map[string]int64
}

type actEntry struct {
	Address     string `json:"address,omitempty"`
	BucketID    string `json:"account_id,omitempty"`
	BucketLabel string `json:"account_label,omitempty"`

	Amount     int64  `json:"amount"`
	AssetID    string `json:"asset_id"`
	AssetLabel string `json:"asset_label"`
}

type actEntryOrder []actEntry

func (a actEntryOrder) Len() int      { return len(a) }
func (a actEntryOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a actEntryOrder) Less(i, j int) bool {
	// Show bucket items first
	if a[i].BucketLabel != "" && a[j].BucketLabel == "" {
		return true
	}
	if a[i].BucketLabel == "" && a[j].BucketLabel != "" {
		return false
	}

	// Sort by bucket ID, address, asset ID, and amount
	if a[i].BucketLabel != a[j].BucketLabel {
		return a[i].BucketLabel < a[j].BucketLabel
	}
	if a[i].Address != a[j].Address {
		return a[i].Address < a[j].Address
	}
	if a[i].AssetLabel != a[j].AssetLabel {
		return a[i].AssetLabel < a[j].AssetLabel
	}

	// If coalescing similar assets within the same bucket or address space is
	// successful, we shouldn't ever get here.
	return a[i].Amount < a[j].Amount
}

type actItem struct {
	TxHash  string     `json:"transaction_id"`
	Time    time.Time  `json:"transaction_time"`
	Inputs  []actEntry `json:"inputs"`
	Outputs []actEntry `json:"outputs"`
}

// getActUTXOs returns all UTXOs consumed by the specified inputs of a
// transaction.
func getActUTXOs(ctx context.Context, txHashes []string, indexes []uint32) ([]*actUTXO, error) {
	q := `
		WITH outpoints AS (
			SELECT unnest($1::text[]), unnest($2::int[])
		)
		SELECT asset_id, amount,
			address_id, bucket_id, wallet_id
		FROM utxos
		WHERE (txid, index) IN (TABLE outpoints)
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(txHashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*actUTXO
	for rows.Next() {
		utxo := new(actUTXO)

		err := rows.Scan(
			&utxo.assetID, &utxo.amount,
			&utxo.addrID, &utxo.bucketID, &utxo.walletID,
		)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}

		res = append(res, utxo)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return res, nil
}

// getActUTXOsByTx retrieves all UTXOs in the specified transaction.
func getActUTXOsByTx(ctx context.Context, txHash string) ([]*actUTXO, error) {
	q := `
		SELECT asset_id, amount,
			address_id, bucket_id, wallet_id
		FROM utxos
		WHERE txid = $1
	`
	rows, err := pg.FromContext(ctx).Query(q, txHash)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*actUTXO
	for rows.Next() {
		utxo := new(actUTXO)

		err := rows.Scan(
			&utxo.assetID, &utxo.amount,
			&utxo.addrID, &utxo.bucketID, &utxo.walletID,
		)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}

		res = append(res, utxo)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return res, nil
}

// getIDsFromUTXOs extracts lists of unique identifiers present from the
// specified UTXOs. It is useful for determining the range of resources involved
// in a transaction.
func getIDsFromUTXOs(utxos []*actUTXO) (
	assetIDs []string, // list of unique asset IDs
	addrIDs []string, // list of unique address IDs
	bucketIDs []string, // list of unique bucket IDs
	walletIDs []string, // list of unique wallet IDs
	walletBuckets map[string][]string, // map of wallet IDs to unique bucket IDs
) {
	walletBuckets = make(map[string][]string)
	for _, u := range utxos {
		assetIDs = append(assetIDs, u.assetID)
		addrIDs = append(addrIDs, u.addrID)
		bucketIDs = append(bucketIDs, u.bucketID)
		walletIDs = append(walletIDs, u.walletID)
		walletBuckets[u.walletID] = append(walletBuckets[u.walletID], u.bucketID)
	}

	sort.Strings(assetIDs)
	sort.Strings(addrIDs)
	sort.Strings(bucketIDs)
	sort.Strings(walletIDs)

	assetIDs = strings.Uniq(assetIDs)
	addrIDs = strings.Uniq(addrIDs)
	bucketIDs = strings.Uniq(bucketIDs)
	walletIDs = strings.Uniq(walletIDs)

	for walletID, buckets := range walletBuckets {
		sort.Strings(buckets)
		walletBuckets[walletID] = strings.Uniq(buckets)
	}

	return assetIDs, addrIDs, bucketIDs, walletIDs, walletBuckets
}

func getActAddrs(ctx context.Context, addrIDs []string) ([]*actAddr, error) {
	q := `
		SELECT id, address, is_change
		FROM addresses
		WHERE id = ANY($1)
		ORDER BY id
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(addrIDs))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*actAddr
	for rows.Next() {
		a := new(actAddr)
		err := rows.Scan(&a.id, &a.address, &a.isChange)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		res = append(res, a)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return res, nil
}

func getActAssets(ctx context.Context, assetIDs []string) ([]*actAsset, error) {
	q := `
		SELECT a.id, a.label, ag.id, ag.application_id
		FROM assets a
		JOIN asset_groups ag ON a.asset_group_id = ag.id
		WHERE a.id = ANY($1)
		ORDER BY a.id
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(assetIDs))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*actAsset
	for rows.Next() {
		a := new(actAsset)
		err := rows.Scan(&a.id, &a.label, &a.agID, &a.appID)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		res = append(res, a)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return res, nil
}

func getActBuckets(ctx context.Context, bucketIDs []string) ([]*actBucket, error) {
	q := `
		SELECT b.id, b.label, b.wallet_id, w.application_id
		FROM buckets b
		JOIN wallets w ON b.wallet_id = w.id
		WHERE b.id = ANY($1)
		ORDER BY b.id
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(bucketIDs))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*actBucket
	for rows.Next() {
		b := new(actBucket)
		err := rows.Scan(&b.id, &b.label, &b.walletID, &b.appID)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		res = append(res, b)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return res, nil
}

func coalesceActivity(ins, outs []*actUTXO, visibleBuckets, changeAddrs []string) txRawActivity {
	// create lookup tables for bucket visibility and change addresses
	isBucketVis := make(map[string]bool)
	for _, bid := range visibleBuckets {
		isBucketVis[bid] = true
	}

	isChange := make(map[string]bool)
	for _, aid := range changeAddrs {
		isChange[aid] = true
	}

	res := txRawActivity{
		insByA:  make(map[string]map[string]int64),
		insByB:  make(map[string]map[string]int64),
		outsByA: make(map[string]map[string]int64),
		outsByB: make(map[string]map[string]int64),
	}

	// Pool all inputs by address, or by bucket if the bucket is visible.
	for _, u := range ins {
		if isBucketVis[u.bucketID] {
			if res.insByB[u.bucketID] == nil {
				res.insByB[u.bucketID] = make(map[string]int64)
			}
			res.insByB[u.bucketID][u.assetID] += u.amount
		} else {
			if res.insByA[u.addrID] == nil {
				res.insByA[u.addrID] = make(map[string]int64)
			}
			res.insByA[u.addrID][u.assetID] += u.amount
		}
	}

	// Pool all outputs by address, or by bucket if the bucket is visible.
	for _, u := range outs {
		if isBucketVis[u.bucketID] {
			// Rather than create a discrete output for a change address, we
			// should deduct the value of the output from the corresponding
			// value in the input. To determine whether to do this, we'll use
			// the following heuristics:
			// 1. The output is paid to a change address.
			// 2. There is a corresponding input for the same bucket and asset.
			// 3. The input's value is greater than the output.

			if isChange[u.addrID] &&
				res.insByB[u.bucketID] != nil &&
				res.insByB[u.bucketID][u.assetID] > u.amount {
				res.insByB[u.bucketID][u.assetID] -= u.amount
			} else {
				if res.outsByB[u.bucketID] == nil {
					res.outsByB[u.bucketID] = make(map[string]int64)
				}
				res.outsByB[u.bucketID][u.assetID] += u.amount
			}
		} else {
			if res.outsByA[u.addrID] == nil {
				res.outsByA[u.addrID] = make(map[string]int64)
			}
			res.outsByA[u.addrID][u.assetID] += u.amount
		}
	}

	return res
}

// createActEntries takes coalesced activity entries and replaces address IDs
// with addresses, and attaches asset and bucket labels. It ensures the result
// is sorted in a consistent order.
func createActEntries(
	r txRawActivity,
	addrs map[string]string,
	assetLabels map[string]string,
	bucketLabels map[string]string,
) (ins, outs []actEntry) {
	for addrID, assetAmts := range r.insByA {
		for assetID, amt := range assetAmts {
			ins = append(ins, actEntry{
				Address:    addrs[addrID],
				AssetID:    assetID,
				AssetLabel: assetLabels[assetID],
				Amount:     amt,
			})
		}
	}

	for bucketID, assetAmts := range r.insByB {
		for assetID, amt := range assetAmts {
			ins = append(ins, actEntry{
				BucketID:    bucketID,
				BucketLabel: bucketLabels[bucketID],
				AssetID:     assetID,
				AssetLabel:  assetLabels[assetID],
				Amount:      amt,
			})
		}
	}

	for addrID, assetAmts := range r.outsByA {
		for assetID, amt := range assetAmts {
			outs = append(outs, actEntry{
				Address:    addrs[addrID],
				AssetID:    assetID,
				AssetLabel: assetLabels[assetID],
				Amount:     amt,
			})
		}
	}

	for bucketID, assetAmts := range r.outsByB {
		for assetID, amt := range assetAmts {
			outs = append(outs, actEntry{
				BucketID:    bucketID,
				BucketLabel: bucketLabels[bucketID],
				AssetID:     assetID,
				AssetLabel:  assetLabels[assetID],
				Amount:      amt,
			})
		}
	}

	sort.Sort(actEntryOrder(ins))
	sort.Sort(actEntryOrder(outs))

	return ins, outs
}

// TODO(jeffomatic): This is identical to asset.isIssuance, but is copied here
// to avoid circular dependencies betwen the two packages. This should probably
// be moved to the fedchain(-sandbox?)/wire package at some point.
func isIssuance(msg *wire.MsgTx) bool {
	emptyHash := wire.Hash32{}
	if len(msg.TxIn) == 1 && msg.TxIn[0].PreviousOutPoint.Hash == emptyHash {
		if len(msg.TxOut) == 0 {
			return false
		}
		assetID := msg.TxOut[0].AssetID
		for _, out := range msg.TxOut {
			if out.AssetID != assetID {
				return false
			}
		}
		return true
	}
	return false
}

func serializeActvity(txHash string, txTime time.Time, ins, outs []actEntry) ([]byte, error) {
	return json.Marshal(actItem{
		TxHash:  txHash,
		Time:    txTime.UTC(),
		Inputs:  ins,
		Outputs: outs,
	})
}

func writeWalletActivity(ctx context.Context, walletID, txHash string, data []byte, bucketIDs []string) error {
	aq := `
		INSERT INTO activity (wallet_id, txid, data)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := pg.FromContext(ctx).QueryRow(aq, walletID, txHash, data).Scan(&id)
	if err != nil {
		return errors.Wrap(err, "insert activity")
	}

	bucketq := `
		INSERT INTO activity_buckets (activity_id, bucket_id)
		VALUES ($1, unnest($2::text[]))
	`
	_, err = pg.FromContext(ctx).Exec(bucketq, id, pg.Strings(bucketIDs))
	if err != nil {
		return errors.Wrap(err, "insert activity for bucket")
	}

	return nil
}

func writeIssuanceActivity(ctx context.Context, a *actAsset, txHash string, data []byte) error {
	iaq := `
		INSERT INTO issuance_activity (asset_group_id, txid, data)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := pg.FromContext(ctx).QueryRow(iaq, a.agID, txHash, data).Scan(&id)
	if err != nil {
		return errors.Wrap(err, "insert issuance activity")
	}

	assetq := `
		INSERT INTO issuance_activity_assets (issuance_activity_id, asset_id)
		VALUES ($1, $2)
	`
	_, err = pg.FromContext(ctx).Exec(assetq, id, a.id)
	if err != nil {
		return errors.Wrap(err, "insert issuance activity for asset")
	}

	return nil
}
