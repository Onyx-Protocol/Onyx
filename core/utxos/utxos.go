package utxos

import (
	"context"

	"github.com/lib/pq"

	"chain/core/pin"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

const (
	PinName       = "utxos"
	DeletePinName = "delete-spent-utxos"
)

type Store struct {
	DB       pg.DB
	PinStore *pin.Store
	Chain    *protocol.Chain
}

func (s *Store) ProcessBlocks(ctx context.Context) {
	if s.PinStore == nil {
		return
	}
	go s.PinStore.ProcessBlocks(ctx, s.Chain, DeletePinName, func(ctx context.Context, b *legacy.Block) error {
		<-s.PinStore.PinWaiter(PinName, b.Height)
		return s.deleteSpent(ctx, b)
	})
	s.PinStore.ProcessBlocks(ctx, s.Chain, PinName, s.index)
}

func (s *Store) deleteSpent(ctx context.Context, b *legacy.Block) error {
	var outputIDs [][]byte
	for _, tx := range b.Transactions {
		for _, inpID := range tx.Tx.InputIDs {
			if sp, err := tx.Spend(inpID); err == nil {
				outputIDs = append(outputIDs, sp.SpentOutputId.Bytes())
			}
		}
	}
	const delQ = `
		DELETE FROM utxos
		WHERE output_id IN (SELECT unnest($1::bytea[]))
	`
	_, err := s.DB.ExecContext(ctx, delQ, pq.ByteaArray(outputIDs))
	return errors.Wrap(err, "deleting spent account utxos")
}

func (s *Store) index(ctx context.Context, b *legacy.Block) error {
	var (
		outputID  pq.ByteaArray
		assetID   pq.ByteaArray
		amount    pq.Int64Array
		program   pq.ByteaArray
		sourceID  pq.ByteaArray
		sourcePos pq.Int64Array
		refData   pq.ByteaArray
	)
	for _, tx := range b.Transactions {
		for j, out := range tx.Outputs {
			resOutID := tx.ResultIds[j]
			resOut, ok := tx.Entries[*resOutID].(*bc.Output)
			if !ok {
				continue
			}
			outputID = append(outputID, tx.OutputID(j).Bytes())
			assetID = append(assetID, out.AssetAmount.AssetId.Bytes())
			amount = append(amount, int64(out.AssetAmount.Amount))
			program = append(program, out.ControlProgram)
			sourceID = append(sourceID, resOut.Source.Ref.Bytes())
			sourcePos = append(sourcePos, int64(resOut.Source.Position))
			refData = append(refData, resOut.Data.Bytes())
		}
	}

	const q = `
		INSERT INTO utxos (output_id, asset_id, amount, control_program,
			source_id, source_pos, ref_data_hash)
		SELECT unnest($1::bytea[]), unnest($2::bytea[]),  unnest($3::bigint[]),
			   unnest($4::bytea[]), unnest($5::bytea[]), unnest($6::bigint[]), unnest($7::bytea[])
		ON CONFLICT (output_id) DO NOTHING
	`
	_, err := s.DB.ExecContext(ctx, q,
		outputID,
		assetID,
		amount,
		program,
		sourceID,
		sourcePos,
		refData,
	)
	return errors.Wrap(err)
}
