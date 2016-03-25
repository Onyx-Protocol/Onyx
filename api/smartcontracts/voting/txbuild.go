package voting

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
)

// RightIssuance builds a txbuilder Receiver issuance for an asset that
// is being issued into a voting right contract.
func RightIssuance(ctx context.Context, holderScript []byte) txbuilder.Receiver {
	return rightScriptData{
		HolderScript:   holderScript,
		Delegatable:    true,
		Deadline:       infiniteDeadline,
		OwnershipChain: bc.Hash{},
	}
}

// RightAuthentication builds txbuilder Reserver and Receiver implementations
// for passing a voting right token through a transaction unchanged. The
// output voting right is identical to the input voting right. Its
// presence in the transaction proves voting right ownership during
// voting.
func RightAuthentication(ctx context.Context, src *RightWithUTXO) (txbuilder.Reserver, txbuilder.Receiver, error) {
	originalHolderAddr, err := appdb.GetAddress(ctx, src.HolderScript)
	if err != nil {
		holderScriptStr, _ := txscript.DisasmString(src.HolderScript)
		return nil, nil, errors.Wrapf(err, "could not get address for holder script [%s]", holderScriptStr)
	}

	reserver := rightsReserver{
		outpoint:   src.Outpoint,
		clause:     clauseAuthenticate,
		output:     src.rightScriptData, // unchanged
		holderAddr: originalHolderAddr,
	}
	return reserver, reserver.output, nil
}

// RightTransfer builds txbuilder Reserver and Receiver implementations for
// a voting right transfer.
func RightTransfer(ctx context.Context, src *RightWithUTXO, newHolderScript []byte) (txbuilder.Reserver, txbuilder.Receiver, error) {
	currentHolderAddr, err := appdb.GetAddress(ctx, src.HolderScript)
	if err != nil {
		holderScriptStr, _ := txscript.DisasmString(src.HolderScript)
		return nil, nil, errors.Wrapf(err, "could not get address for holder script [%s]", holderScriptStr)
	}

	reserver := rightsReserver{
		outpoint: src.Outpoint,
		clause:   clauseTransfer,
		output: rightScriptData{
			HolderScript:   newHolderScript,
			Delegatable:    src.rightScriptData.Delegatable,    // unchanged
			Deadline:       src.rightScriptData.Deadline,       // unchanged
			OwnershipChain: src.rightScriptData.OwnershipChain, // unchanged
		},
		holderAddr: currentHolderAddr,
	}
	return reserver, reserver.output, nil
}

// RightDelegation builds txbuilder Reserver and Receiver implementations for
// delegating a voting right to another party.
func RightDelegation(ctx context.Context, src *RightWithUTXO, newHolderScript []byte, newDeadline int64, delegatable bool) (txbuilder.Reserver, txbuilder.Receiver, error) {
	currentHolderAddr, err := appdb.GetAddress(ctx, src.HolderScript)
	if err != nil {
		holderScriptStr, _ := txscript.DisasmString(src.HolderScript)
		return nil, nil, errors.Wrapf(err, "could not get address for holder script [%s]", holderScriptStr)
	}

	reserver := rightsReserver{
		outpoint: src.Outpoint,
		clause:   clauseDelegate,
		output: rightScriptData{
			HolderScript: newHolderScript,
			Delegatable:  delegatable,
			Deadline:     newDeadline,
			OwnershipChain: calculateOwnershipChain(
				src.OwnershipChain,
				src.HolderScript,
				src.Deadline,
			),
		},
		holderAddr: currentHolderAddr,
	}
	return reserver, reserver.output, nil
}

// RightRecall builds txbuilder Reserver and Receiver implementations for
// a voting right recall.
func RightRecall(ctx context.Context, src, recallPoint *RightWithUTXO, intermediaryRights []*RightWithUTXO) (txbuilder.Reserver, txbuilder.Receiver, error) {
	originalHolderAddr, err := appdb.GetAddress(ctx, recallPoint.HolderScript)
	if err != nil {
		holderScriptStr, _ := txscript.DisasmString(recallPoint.HolderScript)
		return nil, nil, errors.Wrapf(err, "could not get address for holder script [%s]", holderScriptStr)
	}

	intermediaries := make([]intermediateHolder, 0, len(intermediaryRights))
	for _, r := range intermediaryRights {
		intermediaries = append(intermediaries, intermediateHolder{
			script:   r.HolderScript,
			deadline: r.Deadline,
		})
	}

	reserver := rightsReserver{
		outpoint:       src.Outpoint,
		clause:         clauseRecall,
		output:         recallPoint.rightScriptData,
		intermediaries: intermediaries,
		holderAddr:     originalHolderAddr,
	}
	return reserver, reserver.output, nil
}
