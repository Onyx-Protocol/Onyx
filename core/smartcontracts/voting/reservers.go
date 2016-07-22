package voting

import (
	"errors"
	"time"

	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
)

// rightsReserver implements txbuilder.Reserver for assets in the voting
// rights holding contract
type rightsReserver struct {
	outpoint       bc.Outpoint
	clause         rightsContractClause
	output         rightScriptData
	intermediaries []RightHolder // recall clause
	proofHashes    []RightHolder // override clause
	newHolders     []RightHolder // override clause
	forkHash       bc.Hash       // override clause
	prevScript     []byte
	holderAddr     *appdb.Address
	adminAddr      *appdb.Address
}

// RightHolder represents a script with ownership over a
// voting right. When recalling a token, you must provide all intermediate
// holders between the recall point and the current utxo.
type RightHolder struct {
	Script []byte
}

// hash returns Hash256(script). This hash is
// used within the chain of ownership hash chain. When invoking the recall
// clause of the contract, it's necessary to provide these hashes for all
// intermediate holders between the recall holder and the current holder
// to prove prior ownership.
func (rh RightHolder) hash() bc.Hash {
	return sha3.Sum256(rh.Script)
}

// Reserve builds a ReserveResult including the sigscript suffix to satisfy
// the existing UTXO's right holding contract. Reserve satisfies the
// txbuilder.Reserver interface.
func (r rightsReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	var (
		sigscript []*txbuilder.SigScriptComponent
		addrs     []appdb.Address
	)
	if r.holderAddr != nil {
		addrs = append(addrs, *r.holderAddr)
	}
	if r.adminAddr != nil {
		addrs = append(addrs, *r.adminAddr)
	}

	for _, addr := range addrs {
		sigscript = append(sigscript,
			&txbuilder.SigScriptComponent{
				Type:     "signature",
				Required: addr.SigsRequired,
				Signatures: txbuilder.InputSigs(
					hdkey.Derive(addr.Keys, appdb.ReceiverPath(&addr, addr.Index)),
				),
			}, &txbuilder.SigScriptComponent{
				Type: "data",
				Data: addr.RedeemScript,
			})
	}

	var inputs []txscript.Item

	// Add clause-specific parameters:
	switch r.clause {
	case clauseAuthenticate:
		// No clause-specific parameters.
	case clauseTransfer:
		inputs = append(inputs, txscript.DataItem(r.output.HolderScript))
	case clauseDelegate:
		inputs = append(inputs, txscript.BoolItem(r.output.Delegatable))
		inputs = append(inputs, txscript.DataItem(r.output.HolderScript))
	case clauseRecall:
		for _, i := range r.intermediaries {
			h := i.hash()
			inputs = append(inputs, txscript.DataItem(h[:]))
		}
		inputs = append(inputs, txscript.NumItem(int64(len(r.intermediaries))))
		inputs = append(inputs, txscript.DataItem(r.output.HolderScript))
		inputs = append(inputs, txscript.DataItem(r.output.OwnershipChain[:]))
	case clauseOverride:
		for _, h := range r.newHolders {
			inputs = append(inputs, txscript.DataItem(h.Script))
		}
		inputs = append(inputs, txscript.NumItem(len(r.newHolders)))
		for _, ph := range r.proofHashes {
			h := ph.hash()
			inputs = append(inputs, txscript.DataItem(h[:]))
		}
		inputs = append(inputs, txscript.NumItem(len(r.proofHashes)))
		inputs = append(inputs, txscript.DataItem(r.forkHash[:]))
		inputs = append(inputs, txscript.BoolItem(r.output.Delegatable))
	case clauseCancel:
		// TODO(jackson): Implement.
		return nil, errors.New("unimplemented")
	}
	inputs = append(inputs, txscript.NumItem(r.clause))

	programArgs, err := txscript.CheckRedeemP2C(r.prevScript, rightsHoldingContract, inputs)
	if err != nil {
		return nil, err
	}

	for _, arg := range programArgs {
		sigscript = append(sigscript, &txbuilder.SigScriptComponent{
			Type: "data",
			Data: arg,
		})
	}

	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: bc.NewSpendInput(r.outpoint.Hash, r.outpoint.Index, nil, assetAmount.AssetID, assetAmount.Amount, r.prevScript, nil),
				TemplateInput: &txbuilder.Input{
					AssetAmount:   *assetAmount,
					SigComponents: sigscript,
				},
			},
		},
	}
	return result, nil
}

type tokenReserver struct {
	outpoint      bc.Outpoint
	clause        tokenContractClause
	output        tokenScriptData
	registrations []Registration
	distributions map[bc.AssetID]uint64
	rightScript   []byte
	prevScript    []byte
	adminAddr     *appdb.Address
}

// Reserve builds a ReserveResult including the sigscript suffix to satisfy
// the existing UTXO's token holding contract. Reserve satisfies the
// txbuilder.Reserver interface.
func (r tokenReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	var sigscript []*txbuilder.SigScriptComponent

	if r.adminAddr != nil {
		sigscript = append(sigscript,
			&txbuilder.SigScriptComponent{
				Type:     "signature",
				Required: r.adminAddr.SigsRequired,
				Signatures: txbuilder.InputSigs(
					hdkey.Derive(r.adminAddr.Keys, appdb.ReceiverPath(r.adminAddr, r.adminAddr.Index)),
				),
			}, &txbuilder.SigScriptComponent{
				Type: "data",
				Data: r.adminAddr.RedeemScript,
			},
		)
	}

	var inputs []txscript.Item

	switch r.clause {
	case clauseRedistribute:
		for rightAssetID, amount := range r.distributions {
			asset := rightAssetID
			inputs = append(inputs, txscript.NumItem(int64(amount)), txscript.DataItem(asset[:]))
		}
		inputs = append(inputs, txscript.NumItem(int64(len(r.distributions))))
		inputs = append(inputs, txscript.DataItem(r.rightScript))
	case clauseRegister:
		for _, registration := range r.registrations {
			inputs = append(inputs, txscript.NumItem(int64(registration.Amount)), txscript.DataItem(registration.ID))
		}
		inputs = append(inputs, txscript.NumItem(int64(len(r.registrations))))
		inputs = append(inputs, txscript.DataItem(r.rightScript))
	case clauseVote:
		inputs = append(inputs, txscript.NumItem(r.output.Vote))
		inputs = append(inputs, txscript.DataItem(r.rightScript))
	case clauseReset:
		inputs = append(inputs, txscript.NumItem(r.output.State))
	}
	inputs = append(inputs, txscript.NumItem(int64(r.clause)))

	programArgs, err := txscript.CheckRedeemP2C(r.prevScript, tokenHoldingContract, inputs)
	if err != nil {
		return nil, err
	}

	for _, arg := range programArgs {
		sigscript = append(sigscript, &txbuilder.SigScriptComponent{
			Type: "data",
			Data: arg,
		})
	}

	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: bc.NewSpendInput(r.outpoint.Hash, r.outpoint.Index, nil, assetAmount.AssetID, assetAmount.Amount, r.prevScript, nil),
				TemplateInput: &txbuilder.Input{
					AssetAmount:   *assetAmount,
					SigComponents: sigscript,
				},
			},
		},
	}
	return result, nil
}
