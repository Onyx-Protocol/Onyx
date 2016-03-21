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
	originalHolderAddr, err := appdb.GetAddress(ctx, src.HolderScript)
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
		holderAddr: originalHolderAddr,
	}
	return reserver, reserver.output, nil
}
