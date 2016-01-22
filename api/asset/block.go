package asset

import (
	"database/sql"
	"runtime"
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/api/asset/nodetxlog"
	"chain/api/signer"
	"chain/api/txdb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/log"
	"chain/net/rpc"
	"chain/net/trace/span"
)

var fc *fedchain.FC

// Init sets the package level fedchain. If isManager is true,
// Init registers all necessary callbacks for updating
// application state with the fedchain.
func Init(chain *fedchain.FC, signer *signer.Signer, isManager bool) {
	fc = chain
	if isManager {
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
	}
}

// BlockKey is the private key used to sign blocks.
var BlockKey *btcec.PrivateKey

// MakeOrGetBlocks runs forever, attempting to either
// make one block per period, if this is a generator node,
// or get one block per period from a remote generator.
// The caller should call it exactly once.
func MakeOrGetBlocks(ctx context.Context, period time.Duration) {
	for range time.Tick(period) {
		if *Generator == "" {
			makeBlock(ctx)
		} else {
			getBlocks(ctx)
		}
	}
}

// Use of this function must be inside a defer.
func recoverAndLogError(ctx context.Context) {
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
}

func makeBlock(ctx context.Context) {
	defer recoverAndLogError(ctx)
	MakeBlock(ctx, BlockKey)
}

func getBlocks(ctx context.Context) {
	defer recoverAndLogError(ctx)
	store := new(txdb.Store)
	latestBlock, err := store.LatestBlock(ctx)
	if err != nil && errors.Root(err) != sql.ErrNoRows {
		log.Error(ctx, errors.Wrapf(err, "could not fetch latest block"))
	}

	var height *uint64
	if latestBlock != nil {
		height = &latestBlock.Height
	}

	var blocks []*bc.Block
	if err := rpc.Call(ctx, *Generator, "/rpc/generator/get-blocks", height, &blocks); err != nil {
		log.Error(ctx, err)
	}

	for _, b := range blocks {
		err := fc.AddBlock(ctx, b)
		if err != nil {
			log.Error(ctx, errors.Wrapf(err, "applying block at height %d", b.Height))
			return
		}
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
