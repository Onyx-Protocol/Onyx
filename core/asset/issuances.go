package asset

import (
	"golang.org/x/net/context"

	"chain/core/txdb"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/database/pg"
	"chain/errors"
	chainlog "chain/log"
)

// Issuances counts the total units of assets issued and destroyed.
type Issuances struct {
	Assets map[bc.AssetID]IssuanceAmount
}

// IssuanceAmount stores the number of units of an asset issued and destroyed.
type IssuanceAmount struct {
	Issued, Destroyed uint64
}

// Circulation returns all of the issuances for the provided assets
// contained within the blockchain.
func Circulation(ctx context.Context, assetIDs ...bc.AssetID) (Issuances, error) {
	assetIDStrings := make([]string, len(assetIDs))
	for i, assetID := range assetIDs {
		assetIDStrings[i] = assetID.String()
	}

	const q = `
		SELECT asset_id, issued, destroyed
		FROM issuance_totals
		WHERE asset_id IN (SELECT unnest($1::text[]))
	`
	assets := map[bc.AssetID]IssuanceAmount{}
	err := pg.ForQueryRows(ctx, q, pg.Strings(assetIDStrings), func(assetID bc.AssetID, issued, destroyed uint64) {
		assets[assetID] = IssuanceAmount{
			Issued:    issued,
			Destroyed: destroyed,
		}
	})
	return Issuances{Assets: assets}, err
}

// PoolIssuances returns all of the issuances contained within the pending
// tx pool.
func PoolIssuances(ctx context.Context, pool *txdb.Pool) (Issuances, error) {
	// TODO(jackson): Index pool issuances to avoid scanning the entire pool.
	txs, err := pool.Dump(ctx)
	if err != nil {
		return Issuances{}, err
	}
	return calcIssuances(txs...), nil
}

// recordIssuances is a cos block callback that updates the issuance_totals
// table with all issuances within the provided block.
func recordIssuances(ctx context.Context, b *bc.Block, conflicts []*bc.Tx) {
	issuances := calcIssuances(b.Transactions...)

	var (
		assetIDs  = make([]string, 0, len(issuances.Assets))
		issued    = make([]uint64, 0, len(issuances.Assets))
		destroyed = make([]uint64, 0, len(issuances.Assets))
	)
	for assetID, amt := range issuances.Assets {
		assetIDs = append(assetIDs, assetID.String())
		issued = append(issued, amt.Issued)
		destroyed = append(destroyed, amt.Destroyed)
	}
	const updateQ = `
		WITH block_issued AS (
			SELECT * FROM unnest($1::text[], $2::bigint[], $3::bigint[])
			AS t(asset_id, issued, destroyed)
		)
		INSERT INTO issuance_totals(asset_id, height, issued, destroyed)
		SELECT asset_id, $4, issued, destroyed
		FROM block_issued
		ON CONFLICT (asset_id) DO UPDATE
		SET
			height    = excluded.height,
			issued    = issuance_totals.issued + excluded.issued,
			destroyed = issuance_totals.destroyed + excluded.destroyed
		WHERE issuance_totals.height = excluded.height - 1
	`
	_, err := pg.Exec(ctx, updateQ, pg.Strings(assetIDs), pg.Uint64s(issued), pg.Uint64s(destroyed), b.Height)
	if err != nil {
		// TODO(jackson): make this error stop log replay (e.g. crash the process)
		chainlog.Write(ctx, "at", "issuance totals indexing block", "block", b.Height, "error", errors.Wrap(err))
	}
}

func calcIssuances(txs ...*bc.Tx) Issuances {
	assets := map[bc.AssetID]IssuanceAmount{}
	for _, tx := range txs {
		for _, txin := range tx.Inputs {
			if txin.IsIssuance() {
				amt := assets[txin.AssetAmount.AssetID]
				amt.Issued = amt.Issued + txin.AssetAmount.Amount
				assets[txin.AssetAmount.AssetID] = amt
			}
		}

		for _, txout := range tx.Outputs {
			if txscript.IsUnspendable(txout.Script) {
				amt := assets[txout.AssetID]
				amt.Destroyed = amt.Destroyed + txout.Amount
				assets[txout.AssetID] = amt
			}
		}
	}
	return Issuances{Assets: assets}
}
