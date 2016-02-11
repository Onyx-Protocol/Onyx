package asset

import (
	"runtime"
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/api/appdb"
	"chain/api/asset/nodetxlog"
	"chain/api/signer"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/log"
	"chain/net/trace/span"
)

var fc *fedchain.FC

// ConnectFedchain sets the package level fedchain
// as well as registers all necessary callbacks
// with the fedchain.
func ConnectFedchain(chain *fedchain.FC, signer *signer.Signer) {
	// TODO(kr): rename this to Init.
	fc = chain
	fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
		err := addAccountData(ctx, tx)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "adding account data"))
		}
	})
	fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
		err := nodetxlog.Write(ctx, tx, time.Now())
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "writing activitiy"))
		}
	})
	fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
		if tx.IsIssuance() {
			asset, amt := issued(tx.Outputs)
			err := appdb.UpdateIssuances(
				ctx,
				map[bc.AssetID]int64{asset: int64(amt)},
				false,
			)
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "update issuances"))
			}
		}
	})
	fc.AddBlockCallback(func(ctx context.Context, block *bc.Block, conflicts []*bc.Tx) {
		issued := issuedAssets(block.Transactions)
		err := appdb.UpdateIssuances(ctx, issued, true)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "update issuances"))
			return
		}
		deletedIssued := issuedAssets(conflicts)
		for asset := range issued {
			issued[asset] *= -1
		}
		for asset, amt := range deletedIssued {
			issued[asset] = -amt
		}
		err = appdb.UpdateIssuances(ctx, issued, false)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "update pool issuances"))
		}
	})
	fc.AddBlockCallback(func(ctx context.Context, block *bc.Block, conflicts []*bc.Tx) {
		outs, err := getRestorableOutputs(ctx, conflicts)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "getting restorable outs"))
			return
		}

		applyToReserver(ctx, outs)
	})
}

// BlockKey is the private key used to sign blocks.
var BlockKey *btcec.PrivateKey

// MakeBlocks runs forever,
// attempting to make one block per period.
// The caller should call it exactly once.
func MakeBlocks(ctx context.Context, period time.Duration) {
	for range time.Tick(period) {
		makeBlock(ctx)
	}
}

func makeBlock(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Write(ctx,
				log.KeyMessage, "panic",
				log.KeyError, err,
				log.KeyStack, buf,
			)
		}
	}()
	b, err := MakeBlock(ctx, BlockKey)
	if err != nil {
		log.Error(ctx, err)
	} else if b != nil {
		log.Messagef(ctx, "made block %s height %d with %d txs", b.Hash(), b.Height, len(b.Transactions))
	}
}

// MakeBlock creates a new bc.Block and updates the txpool/utxo state.
func MakeBlock(ctx context.Context, key *btcec.PrivateKey) (*bc.Block, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	b, err := fc.GenerateBlock(ctx, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "generate")
	}
	if len(b.Transactions) == 0 {
		return nil, nil // don't bother making an empty block
	}
	err = fedchain.SignBlock(b, key)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}
	err = fc.AddBlock(ctx, b)
	if err != nil {
		return nil, errors.Wrap(err, "apply")
	}
	return b, nil
}

func getRestorableOutputs(ctx context.Context, txs []*bc.Tx) (outs []*txdb.Output, err error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	// TODO(kr): probably should use fedchain.FC instead
	store := new(txdb.Store)

	poolView, err := store.NewPoolViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	bcView, err := store.NewViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, errors.Wrap(err, "load prev outs from conflicting txs")
	}

	// undo conflicting txs in reserver
	view := state.MultiReader(poolView, bcView)
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			o := view.Output(ctx, in.Previous)
			if o == nil || o.Spent {
				continue
			}
			outs = append(outs, &txdb.Output{Output: *o})
		}

		for i, out := range tx.Outputs {
			op := bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}
			outs = append(outs, &txdb.Output{
				Output: state.Output{
					TxOutput: *out,
					Outpoint: op,
					Spent:    true,
				},
			})
		}
	}

	err = loadAccountInfo(ctx, outs)
	if err != nil {
		return nil, errors.Wrap(err, "load conflict out account info")
	}

	return outs, nil
}

func applyToReserver(ctx context.Context, outs []*txdb.Output) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var del, ins []*utxodb.UTXO
	for _, out := range outs {
		u := &utxodb.UTXO{
			AccountID: out.AccountID,
			AssetID:   out.AssetID,
			Amount:    out.Amount,
			Outpoint:  out.Outpoint,
			AddrIndex: out.AddrIndex,
		}
		if out.Spent {
			del = append(del, u)
		} else {
			ins = append(ins, u)
		}
	}
	utxoDB.Apply(del, ins)
}

// loadAccountInfoFromAddrs queries the addresses table
// to load account information using output scripts
func loadAccountInfoFromAddrs(ctx context.Context, outs map[bc.Outpoint]*txdb.Output) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		scripts           [][]byte
		outpointsByScript = make(map[string]bc.Outpoint)
	)
	for _, out := range outs {

		scripts = append(scripts, out.Script)
		outpointsByScript[string(out.Script)] = out.Outpoint
	}

	const addrq = `
		SELECT pk_script, manager_node_id, account_id, key_index(key_index)
		FROM addresses
		WHERE pk_script IN (SELECT unnest($1::bytea[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, addrq, pg.Byteas(scripts))
	if err != nil {
		return errors.Wrap(err, "addresses select query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			script         []byte
			mnodeID, accID string
			addrIndex      []uint32
		)
		err := rows.Scan(&script, &mnodeID, &accID, (*pg.Uint32s)(&addrIndex))
		if err != nil {
			return errors.Wrap(err, "addresses row scan")
		}
		out := outs[outpointsByScript[string(script)]]
		out.ManagerNodeID = mnodeID
		out.AccountID = accID
		copy(out.AddrIndex[:], addrIndex)
	}
	return errors.Wrap(rows.Err(), "addresses end row scan loop")
}

// loadAccountInfoFromUTXOs loads account data from the utxos table
// using outpoints
func loadAccountInfoFromUTXOs(ctx context.Context, outs map[bc.Outpoint]*txdb.Output) error {
	var (
		hashes  []string
		indexes []uint32
	)
	for op := range outs {
		hashes = append(hashes, op.Hash.String())
		indexes = append(indexes, op.Index)
	}
	const utxoq = `
		SELECT tx_hash, index, manager_node_id, account_id, key_index(addr_index)
		FROM account_utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, utxoq, pg.Strings(hashes), pg.Uint32s(indexes))
	if err != nil {
		return errors.Wrap(err, "utxos select query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			op             bc.Outpoint
			mnodeID, accID string
			addrIndex      []uint32
		)
		err := rows.Scan(&op.Hash, &op.Index, &mnodeID, &accID, (*pg.Uint32s)(&addrIndex))
		if err != nil {
			return errors.Wrap(err, "utxos row scan")
		}
		out := outs[op]
		out.ManagerNodeID = mnodeID
		out.AccountID = accID
		copy(out.AddrIndex[:], addrIndex)
	}
	return errors.Wrap(rows.Err(), "utxos end row scan loop")
}

// loadAccountInfo returns annotated UTXO data (outputs + account mappings) for
// addresses known to this manager node. It is only concerned with outputs that
// actually have account mappings, which come from either the utxos or
// addresses tables.
func loadAccountInfo(ctx context.Context, outs []*txdb.Output) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	outputs := make(map[bc.Outpoint]*txdb.Output)
	for _, out := range outs {
		outputs[out.Outpoint] = out
	}

	err := loadAccountInfoFromAddrs(ctx, outputs)
	if err != nil {
		return err
	}

	err = loadAccountInfoFromUTXOs(ctx, outputs)
	if err != nil {
		return err
	}

	return nil
}

func issuedAssets(txs []*bc.Tx) map[bc.AssetID]int64 {
	issued := make(map[bc.AssetID]int64)
	for _, tx := range txs {
		if !tx.IsIssuance() {
			continue
		}
		for _, out := range tx.Outputs {
			issued[out.AssetID] += int64(out.Amount)
		}
	}
	return issued
}

// UpsertGenesisBlock creates a genesis block iff it does not exist.
func UpsertGenesisBlock(ctx context.Context) (*bc.Block, error) {
	script, err := fedchain.GenerateBlockScript([]*btcec.PublicKey{BlockKey.PubKey()}, 1)
	if err != nil {
		return nil, err
	}

	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:      bc.NewBlockVersion,
			Timestamp:    uint64(time.Now().Unix()),
			OutputScript: script,
		},
	}

	const q = `
		INSERT INTO blocks (block_hash, height, data, header)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM blocks WHERE height=$2)
	`
	_, err = pg.FromContext(ctx).Exec(ctx, q, b.Hash(), b.Height, b, &b.BlockHeader)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return b, nil
}
