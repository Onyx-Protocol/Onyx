package appdb

import (
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	"golang.org/x/net/context"

	"chain/api/utxodb"
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
func WriteActivity(ctx context.Context, tx *wire.MsgTx, outs []*UTXO, txTime time.Time) error {
	defer metrics.RecordElapsed(time.Now())
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction

	txHash := tx.TxSha().String()

	// Get detailed UTXO information for the transaction's inputs and outputs.
	var (
		hashes  []string
		indexes []uint32
	)
	for _, in := range tx.TxIn {
		hashes = append(hashes, in.PreviousOutPoint.Hash.String())
		indexes = append(indexes, in.PreviousOutPoint.Index)
	}
	ins, err := getActUTXOs(ctx, hashes, indexes)
	if err != nil {
		return errors.Wrap(err, "get input utxos")
	}

	// Extract IDs for all resources involved in the transaction. The lists
	// should not contain duplicates.
	allUTXOs := append(ins, outs...)
	assetIDs, bucketIDs, walletIDs, walletBuckets := getIDsFromUTXOs(allUTXOs)

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
		r := coalesceActivity(ins, outs, walletBuckets[walletID])
		inAct, outAct := createActEntries(r, assetLabels, bucketLabels)

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
			if b.projID == actAssets[0].projID {
				visibleBuckets = append(visibleBuckets, b.id)
			}
		}

		r := coalesceActivity(ins, outs, visibleBuckets)
		inAct, outAct := createActEntries(r, assetLabels, bucketLabels)

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
		WHERE manager_node_id=$1 AND (($2 = '') OR (id < $2))
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
		LEFT JOIN activity_accounts AS aa
		ON a.id=aa.activity_id
		WHERE aa.account_id=$1 AND (($2 = '') OR (a.id < $2))
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
		WHERE issuer_node_id = $1 AND (($2 = '') OR (id < $2))
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
		WHERE manager_node_id=$1 AND txid=$2
	`

	var a []byte
	err := pg.FromContext(ctx).QueryRow(q, walletID, txID).Scan(&a)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "transaction id: %v", txID)
	}
	return (*json.RawMessage)(&a), err
}

// UTXO contains fields we need to insert into the db
// but don't store in memory in package utxodb.
type UTXO struct {
	*utxodb.UTXO
	Addr     string
	WalletID string
	IsChange bool
}

type actAddr struct {
	id       string
	address  string
	isChange bool
}

type actAsset struct {
	id     string
	label  string
	agID   string
	projID string
}

type actBucket struct {
	id       string
	label    string
	walletID string
	projID   string
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
func getActUTXOs(ctx context.Context, txHashes []string, indexes []uint32) ([]*UTXO, error) {
	q := `
		WITH outpoints AS (SELECT unnest($1::text[]), unnest($2::int[]))
		SELECT asset_id, amount, key_index(addr_index), account_id, manager_node_id
		FROM utxos
		WHERE (txid, index) IN (TABLE outpoints)
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(txHashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*UTXO
	var addrIndexes [][]uint32
	for rows.Next() {
		var addrIndex []uint32
		utxo := &UTXO{UTXO: new(utxodb.UTXO)}

		err := rows.Scan(
			&utxo.AssetID, &utxo.Amount,
			(*pg.Uint32s)(&addrIndex), &utxo.BucketID, &utxo.WalletID,
		)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}

		res = append(res, utxo)
		addrIndexes = append(addrIndexes, addrIndex)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "end row scan loop")
	}

	for i, utxo := range res {
		utxo.Addr, err = DeriveAddress(ctx, utxo.BucketID, addrIndexes[i])
		if err != nil {
			return nil, errors.Wrap(err, "derive address")
		}
	}

	return res, nil
}

// getIDsFromUTXOs extracts lists of unique identifiers present from the
// specified UTXOs. It is useful for determining the range of resources involved
// in a transaction.
func getIDsFromUTXOs(utxos []*UTXO) (
	assetIDs []string, // list of unique asset IDs
	bucketIDs []string, // list of unique bucket IDs
	walletIDs []string, // list of unique wallet IDs
	walletBuckets map[string][]string, // map of wallet IDs to unique bucket IDs
) {
	walletBuckets = make(map[string][]string)
	for _, u := range utxos {
		if u != nil {
			assetIDs = append(assetIDs, u.AssetID)
			bucketIDs = append(bucketIDs, u.BucketID)
			walletIDs = append(walletIDs, u.WalletID)
			walletBuckets[u.WalletID] = append(walletBuckets[u.WalletID], u.BucketID)
		}
	}

	sort.Strings(assetIDs)
	sort.Strings(bucketIDs)
	sort.Strings(walletIDs)

	assetIDs = strings.Uniq(assetIDs)
	bucketIDs = strings.Uniq(bucketIDs)
	walletIDs = strings.Uniq(walletIDs)

	for walletID, buckets := range walletBuckets {
		sort.Strings(buckets)
		walletBuckets[walletID] = strings.Uniq(buckets)
	}

	return assetIDs, bucketIDs, walletIDs, walletBuckets
}

func getActAssets(ctx context.Context, assetIDs []string) ([]*actAsset, error) {
	q := `
		SELECT a.id, a.label, i.id, i.project_id
		FROM assets a
		JOIN issuer_nodes i ON a.issuer_node_id = i.id
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
		err := rows.Scan(&a.id, &a.label, &a.agID, &a.projID)
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
		SELECT acc.id, acc.label, acc.manager_node_id, mn.project_id
		FROM accounts acc
		JOIN manager_nodes mn ON acc.manager_node_id = mn.id
		WHERE acc.id = ANY($1)
		ORDER BY acc.id
	`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(bucketIDs))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*actBucket
	for rows.Next() {
		b := new(actBucket)
		err := rows.Scan(&b.id, &b.label, &b.walletID, &b.projID)
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

func coalesceActivity(ins, outs []*UTXO, visibleBuckets []string) txRawActivity {
	// create lookup tables for bucket visibility and change addresses
	isBucketVis := make(map[string]bool)
	for _, bid := range visibleBuckets {
		isBucketVis[bid] = true
	}

	res := txRawActivity{
		insByA:  make(map[string]map[string]int64),
		insByB:  make(map[string]map[string]int64),
		outsByA: make(map[string]map[string]int64),
		outsByB: make(map[string]map[string]int64),
	}

	// Pool all inputs by address, or by bucket if the bucket is visible.
	for _, u := range ins {
		if isBucketVis[u.BucketID] {
			if res.insByB[u.BucketID] == nil {
				res.insByB[u.BucketID] = make(map[string]int64)
			}
			res.insByB[u.BucketID][u.AssetID] += int64(u.Amount)
		} else {
			if res.insByA[u.Addr] == nil {
				res.insByA[u.Addr] = make(map[string]int64)
			}
			res.insByA[u.Addr][u.AssetID] += int64(u.Amount)
		}
	}

	// Pool all outputs by address, or by bucket if the bucket is visible.
	for _, u := range outs {
		if isBucketVis[u.BucketID] {
			// Rather than create a discrete output for a change address, we
			// should deduct the value of the output from the corresponding
			// value in the input. To determine whether to do this, we'll use
			// the following heuristics:
			// 1. The output is paid to a change address.
			// 2. There is a corresponding input for the same bucket and asset.
			// 3. The input's value is greater than the output.

			if u.IsChange &&
				res.insByB[u.BucketID] != nil &&
				res.insByB[u.BucketID][u.AssetID] > int64(u.Amount) {
				res.insByB[u.BucketID][u.AssetID] -= int64(u.Amount)
			} else {
				if res.outsByB[u.BucketID] == nil {
					res.outsByB[u.BucketID] = make(map[string]int64)
				}
				res.outsByB[u.BucketID][u.AssetID] += int64(u.Amount)
			}
		} else {
			if res.outsByA[u.Addr] == nil {
				res.outsByA[u.Addr] = make(map[string]int64)
			}
			res.outsByA[u.Addr][u.AssetID] += int64(u.Amount)
		}
	}

	return res
}

// createActEntries takes coalesced activity entries and replaces address IDs
// with addresses, and attaches asset and bucket labels. It ensures the result
// is sorted in a consistent order.
func createActEntries(
	r txRawActivity,
	assetLabels map[string]string,
	bucketLabels map[string]string,
) (ins, outs []actEntry) {
	for addr, assetAmts := range r.insByA {
		for assetID, amt := range assetAmts {
			ins = append(ins, actEntry{
				Address:    addr,
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

	for addr, assetAmts := range r.outsByA {
		for assetID, amt := range assetAmts {
			outs = append(outs, actEntry{
				Address:    addr,
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
	if ins == nil {
		ins = []actEntry{}
	}
	if outs == nil {
		outs = []actEntry{}
	}

	return json.Marshal(actItem{
		TxHash:  txHash,
		Time:    txTime.UTC(),
		Inputs:  ins,
		Outputs: outs,
	})
}

func writeWalletActivity(ctx context.Context, walletID, txHash string, data []byte, bucketIDs []string) error {
	aq := `
		INSERT INTO activity (manager_node_id, txid, data)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := pg.FromContext(ctx).QueryRow(aq, walletID, txHash, data).Scan(&id)
	if err != nil {
		return errors.Wrap(err, "insert activity")
	}

	bucketq := `
		INSERT INTO activity_accounts (activity_id, account_id)
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
		INSERT INTO issuance_activity (issuer_node_id, txid, data)
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
