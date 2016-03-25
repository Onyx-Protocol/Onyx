package voting

import (
	"errors"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/fedchain/bc"
	"chain/fedchain/hdkey"
	"chain/fedchain/txscript"
)

// rightsReserver implements txbuilder.Reserver for assets in the voting
// rights holding contract
type rightsReserver struct {
	outpoint   bc.Outpoint
	clause     rightsContractClause
	output     rightScriptData
	holderAddr *appdb.Address
}

// Reserve builds a ReserveResult including the sigscript suffix to satisfy
// the existing UTXO's right holding contract. Reserve satisfies the
// txbuilder.Reserver interface.
func (r rightsReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	// TODO(jackson): Include admin redeem script and admin signatures once
	//                the contract supports admin scripts.

	sb := txscript.NewScriptBuilder()

	// Add clause-specific parameters:
	switch r.clause {
	case clauseTransfer:
		sb = sb.
			AddData(r.holderAddr.RedeemScript).
			AddData(r.output.HolderScript)
	case clauseDelegate:
		sb = sb.
			AddData(r.holderAddr.RedeemScript).
			AddInt64(r.output.Deadline).
			AddBool(r.output.Delegatable).
			AddData(r.output.HolderScript)
	case clauseAuthenticate, clauseRecall, clauseOverride, clauseCancel:
		// TODO(jackson): Implement.
		return nil, errors.New("unimplemented")
	}

	sb = sb.
		AddInt64(int64(r.clause)).
		AddData(rightsHoldingContract)

	sigScriptSuffix, err := sb.Script()
	if err != nil {
		return nil, err
	}

	// Build the signatures required for this transaction.
	var (
		signatures []*txbuilder.Signature
		holderKeys = hdkey.Derive(
			r.holderAddr.Keys,
			appdb.ReceiverPath(r.holderAddr, r.holderAddr.Index),
		)
	)
	if r.holderAddr != nil {
		for _, k := range holderKeys {
			signatures = append(signatures, &txbuilder.Signature{
				XPub:           k.Root.String(),
				DerivationPath: k.Path,
			})
		}
	}
	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{
					Previous: r.outpoint,
				},
				TemplateInput: &txbuilder.Input{
					AssetAmount:     *assetAmount,
					SigScriptSuffix: sigScriptSuffix,
					Sigs:            signatures,
				},
			},
		},
	}
	return result, nil
}
