package account

import (
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/net/trace/span"
)

var fc *cos.FC

// Init sets the package level cos. If isManager is true,
// Init registers all necessary callbacks for updating
// application state with the cos.
func Init(chain *cos.FC) {
	if fc == chain {
		// Silently ignore duplicate calls.
		return
	}

	fc = chain
	fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
		err := addAccountData(ctx, tx)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "adding account data"))
		}
	})
	fc.AddBlockCallback(func(ctx context.Context, b *bc.Block) {
		indexAccountUTXOs(ctx, b)
	})
}

type output struct {
	state.Output
	AccountID string
	keyIndex  [2]uint32
}

// Note, FC guarantees it will call the tx callback
// for every tx in b before we get here.
func indexAccountUTXOs(ctx context.Context, b *bc.Block) {
	var (
		pos    []int32
		txhash []string
	)
	for i, tx := range b.Transactions {
		pos = append(pos, int32(i))
		txhash = append(txhash, tx.Hash.String())
	}

	const q = `
		UPDATE account_utxos SET confirmed_in=$3, block_timestamp=$4, block_pos=pos
		FROM (SELECT unnest($1::text[]) AS txhash, unnest($2::integer[]) AS pos) t
		WHERE tx_hash=txhash
	`
	_, err := pg.Exec(
		ctx,
		q,
		pg.Strings(txhash),
		pg.Int32s(pos),
		b.Height,
		b.TimestampMS,
	)
	if err != nil {
		// TODO(kr): make these errors stop log replay (e.g. crash the process)
		log.Write(ctx, "at", "account utxos indexing block", "block", b.Height, "error", errors.Wrap(err))
	}

	deltxhash, delindex := prevoutDBKeys(b.Transactions...)

	const delQ = `
		DELETE FROM account_utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err = pg.Exec(ctx, delQ, deltxhash, delindex)
	if err != nil {
		log.Write(ctx, "block", b.Height, "error", errors.Wrap(err))
		panic(err)
	}
}

func addAccountData(ctx context.Context, tx *bc.Tx) error {
	var outs []*state.Output
	for i, out := range tx.Outputs {
		stateOutput := &state.Output{
			TxOutput: *out,
			Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
		}
		outs = append(outs, stateOutput)
	}

	accOuts, err := loadAccountInfo(ctx, outs)
	if err != nil {
		return errors.Wrap(err, "loading account info from addresses")
	}

	err = insertAccountOutputs(ctx, accOuts)
	return errors.Wrap(err, "updating pool outputs")
}

// loadAccountInfo turns a set of state.Outputs into a set of
// outputs by adding account annotations.  Outputs that can't be
// annotated are excluded from the result.
func loadAccountInfo(ctx context.Context, outs []*state.Output) ([]*output, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	outsByScript := make(map[string][]*state.Output, len(outs))
	for _, out := range outs {
		scriptStr := string(out.ControlProgram)
		outsByScript[scriptStr] = append(outsByScript[scriptStr], out)
	}

	var scripts pg.Byteas
	for s := range outsByScript {
		scripts = append(scripts, []byte(s))
	}

	result := make([]*output, 0, len(outs))

	const q = `
		SELECT signer_id, key_index(key_index), control_program
		FROM account_control_programs
		WHERE control_program IN (SELECT unnest($1::bytea[]))
	`

	err := pg.ForQueryRows(ctx, q, scripts, func(accountID string, keyIndex pg.Uint32s, program []byte) {
		for _, out := range outsByScript[string(program)] {
			newOut := &output{
				Output:    *out,
				AccountID: accountID,
			}
			copy(newOut.keyIndex[:], keyIndex)
			result = append(result, newOut)
		}
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func prevoutDBKeys(txs ...*bc.Tx) (txhash pg.Strings, index pg.Uint32s) {
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			o := in.Outpoint()
			txhash = append(txhash, o.Hash.String())
			index = append(index, o.Index)
		}
	}
	return
}

// insertAccountOutputs records the account data for utxos
func insertAccountOutputs(ctx context.Context, outs []*output) error {
	var (
		txHash    pg.Strings
		index     pg.Uint32s
		assetID   pg.Strings
		amount    pg.Int64s
		accountID pg.Strings
		cpIndex   pg.Int64s
		program   pg.Byteas
		metadata  pg.Byteas
	)
	for _, out := range outs {
		txHash = append(txHash, out.Outpoint.Hash.String())
		index = append(index, out.Outpoint.Index)
		assetID = append(assetID, out.AssetID.String())
		amount = append(amount, int64(out.Amount))
		accountID = append(accountID, out.AccountID)
		cpIndex = append(cpIndex, toKeyIndex(out.keyIndex[:]))
		program = append(program, out.ControlProgram)
		metadata = append(metadata, out.ReferenceData)
	}

	const q = `
		WITH outputs AS (
			SELECT t.* FROM unnest($1::text[], $2::bigint[], $3::text[], $4::bigint[], $5::text[], $6::bigint[], $7::bytea[], $8::bytea[])
			AS t(tx_hash, index, asset_id, amount, acc, control_program_index, control_program, metadata)
		)
		INSERT INTO account_utxos (tx_hash, index, asset_id, amount, account_id, control_program_index, control_program, metadata)
		SELECT * FROM outputs o
		ON CONFLICT (tx_hash, index) DO NOTHING
	`
	_, err := pg.Exec(ctx, q,
		txHash,
		index,
		assetID,
		amount,
		accountID,
		cpIndex,
		program,
		metadata,
	)

	return errors.Wrap(err)
}

func toKeyIndex(i []uint32) int64 {
	return int64(i[0])<<31 | int64(i[1]&0x7fffffff)
}
