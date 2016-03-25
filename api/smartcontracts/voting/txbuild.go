package voting

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/errors"
	"chain/fedchain/txscript"
)

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
