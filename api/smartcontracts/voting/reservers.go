package voting

import (
	"errors"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/crypto/hash256"
	"chain/fedchain/bc"
	"chain/fedchain/hdkey"
	"chain/fedchain/txscript"
)

// rightsReserver implements txbuilder.Reserver for assets in the voting
// rights holding contract
type rightsReserver struct {
	outpoint       bc.Outpoint
	clause         rightsContractClause
	output         rightScriptData
	intermediaries []intermediateHolder
	holderAddr     *appdb.Address
}

// intermediateHolder represents a previous holder. When recalling a token,
// you must provide all intermediate holders between the recall point and the
// current utxo.
type intermediateHolder struct {
	script   []byte
	deadline int64
}

// hash returns Hash256(Hash256(script) + Hash256(deadline)). This hash is
// used within the chain of ownership hash chain. When invoking the recall
// clause of the contract, it's necessary to provide these hashes for all
// intermediate holders between the recall holder and the current holder
// to prove prior ownership.
func (ih intermediateHolder) hash() bc.Hash {
	scriptHash := hash256.Sum(ih.script)
	deadlineHash := hash256.Sum(txscript.Int64ToScriptBytes(ih.deadline))

	data := make([]byte, 0, len(scriptHash)+len(deadlineHash))
	data = append(data, scriptHash[:]...)
	data = append(data, deadlineHash[:]...)
	return hash256.Sum(data)
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
	case clauseAuthenticate:
		sb = sb.
			AddData(r.holderAddr.RedeemScript)
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
	case clauseRecall:
		sb = sb.
			AddData(r.holderAddr.RedeemScript)
		for _, i := range r.intermediaries {
			h := i.hash()
			sb.AddData(h[:])
		}
		sb = sb.
			AddInt64(int64(len(r.intermediaries))).
			AddData(r.output.HolderScript).
			AddInt64(r.output.Deadline).
			AddData(r.output.OwnershipChain[:])
	case clauseOverride, clauseCancel:
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
