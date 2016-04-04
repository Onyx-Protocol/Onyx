package voting

import (
	"errors"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
	"chain/crypto/hash256"
)

// rightsReserver implements txbuilder.Reserver for assets in the voting
// rights holding contract
type rightsReserver struct {
	outpoint       bc.Outpoint
	clause         rightsContractClause
	output         rightScriptData
	intermediaries []intermediateHolder
	prevScript     []byte
	holderAddr     *appdb.Address
	adminAddr      *appdb.Address
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
				Type:   "script",
				Script: txscript.AddDataToScript(nil, addr.RedeemScript),
			})
	}

	// Build up contract-specific sigscript data.
	sb := txscript.NewScriptBuilder()

	// Add clause-specific parameters:
	switch r.clause {
	case clauseAuthenticate:
		// No clause-specific parameters.
	case clauseTransfer:
		sb = sb.
			AddData(r.output.HolderScript)
	case clauseDelegate:
		sb = sb.
			AddInt64(r.output.Deadline).
			AddBool(r.output.Delegatable).
			AddData(r.output.HolderScript)
	case clauseRecall:
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
	script, err := sb.Script()
	if err != nil {
		return nil, err
	}
	sigscript = append(sigscript, &txbuilder.SigScriptComponent{
		Type:   "script",
		Script: script,
	})

	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{
					Previous:    r.outpoint,
					AssetAmount: *assetAmount,
					PrevScript:  r.prevScript,
				},
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
	outpoint    bc.Outpoint
	clause      tokenContractClause
	output      tokenScriptData
	rightScript []byte
	prevScript  []byte
	adminAddr   *appdb.Address
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
				Type:   "script",
				Script: txscript.AddDataToScript(nil, r.adminAddr.RedeemScript),
			},
		)
	}

	sb := txscript.NewScriptBuilder()
	switch r.clause {
	case clauseIntendToVote:
		sb = sb.
			AddData(r.rightScript)
	case clauseVote, clauseFinish, clauseReset:
		// TODO(jackson): Implement.
		return nil, errors.New("unimplemented")
	}
	sb = sb.
		AddInt64(int64(r.clause)).
		AddData(tokenHoldingContract)
	script, err := sb.Script()
	if err != nil {
		return nil, err
	}

	sigscript = append(sigscript, &txbuilder.SigScriptComponent{
		Type:   "script",
		Script: script,
	})

	result := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{
					Previous:    r.outpoint,
					AssetAmount: *assetAmount,
					PrevScript:  r.prevScript,
				},
				TemplateInput: &txbuilder.Input{
					AssetAmount:   *assetAmount,
					SigComponents: sigscript,
				},
			},
		},
	}
	return result, nil
}
