package voting

import (
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/errors"
)

// RightIssuance builds a txbuilder Receiver issuance for an asset that
// is being issued into a voting right contract.
func RightIssuance(ctx context.Context, adminScript, holderScript []byte) txbuilder.Receiver {
	return rightScriptData{
		AdminScript:    adminScript,
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
		prevScript: src.PKScript(),
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
	adminAddr, err := appdb.GetAddress(ctx, src.AdminScript)
	if err != nil {
		adminScriptStr, _ := txscript.DisasmString(src.AdminScript)
		return nil, nil, errors.Wrapf(err, "could not get address for admin script [%s]", adminScriptStr)
	}

	reserver := rightsReserver{
		outpoint: src.Outpoint,
		clause:   clauseTransfer,
		output: rightScriptData{
			AdminScript:    src.AdminScript, // unchanged
			HolderScript:   newHolderScript,
			Delegatable:    src.Delegatable,    // unchanged
			Deadline:       src.Deadline,       // unchanged
			OwnershipChain: src.OwnershipChain, // unchanged
		},
		prevScript: src.PKScript(),
		holderAddr: currentHolderAddr,
		adminAddr:  adminAddr,
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
	adminAddr, err := appdb.GetAddress(ctx, src.AdminScript)
	if err != nil {
		adminScriptStr, _ := txscript.DisasmString(src.AdminScript)
		return nil, nil, errors.Wrapf(err, "could not get address for admin script [%s]", adminScriptStr)
	}

	reserver := rightsReserver{
		outpoint: src.Outpoint,
		clause:   clauseDelegate,
		output: rightScriptData{
			AdminScript:  src.AdminScript,
			HolderScript: newHolderScript,
			Delegatable:  delegatable,
			Deadline:     newDeadline,
			OwnershipChain: calculateOwnershipChain(
				src.OwnershipChain,
				src.HolderScript,
				src.Deadline,
			),
		},
		prevScript: src.PKScript(),
		holderAddr: currentHolderAddr,
		adminAddr:  adminAddr,
	}
	return reserver, reserver.output, nil
}

// RightRecall builds txbuilder Reserver and Receiver implementations for
// a voting right recall.
func RightRecall(ctx context.Context, src, recallPoint *RightWithUTXO, intermediaryRights []*RightWithUTXO) (txbuilder.Reserver, txbuilder.Receiver, error) {
	originalHolderAddr, err := appdb.GetAddress(ctx, recallPoint.HolderScript)
	if err != nil {
		originalHolderAddr = nil
	}
	adminAddr, err := appdb.GetAddress(ctx, src.AdminScript)
	if err != nil {
		adminAddr = nil
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
		prevScript:     src.PKScript(),
		holderAddr:     originalHolderAddr,
		adminAddr:      adminAddr,
	}
	return reserver, reserver.output, nil
}

// TokenIssuance builds a txbuilder Receiver implementation
// for a voting token issuance.
func TokenIssuance(ctx context.Context, rightAssetID bc.AssetID, admin []byte, optionCount int64, secretHash bc.Hash) txbuilder.Receiver {
	scriptData := tokenScriptData{
		Right:       rightAssetID,
		AdminScript: admin,
		OptionCount: optionCount,
		State:       stateDistributed,
		SecretHash:  secretHash,
		Vote:        0,
	}
	return scriptData
}

// TokenRegister builds txbuilder Reserver and Receiver implementations
// for a voting token registration transition.
func TokenRegistration(ctx context.Context, token *Token, rightScript []byte) (txbuilder.Reserver, txbuilder.Receiver, error) {
	prevScript := token.tokenScriptData.PKScript()
	registered := token.tokenScriptData
	registered.State = stateRegistered

	reserver := tokenReserver{
		outpoint:    token.Outpoint,
		clause:      clauseRegister,
		output:      registered,
		prevScript:  prevScript,
		rightScript: rightScript,
	}
	return reserver, registered, nil
}

// TokenVote builds txbuilder Reserver and Receiver implementations
// for a voting token vote transition.
func TokenVote(ctx context.Context, token *Token, rightScript []byte, vote int64, secret []byte) (txbuilder.Reserver, txbuilder.Receiver, error) {
	data := token.tokenScriptData
	data.State = stateVoted
	data.Vote = vote

	reserver := tokenReserver{
		outpoint:    token.Outpoint,
		clause:      clauseVote,
		output:      data,
		prevScript:  token.tokenScriptData.PKScript(),
		rightScript: rightScript,
		secret:      secret,
	}
	return reserver, data, nil
}

// TokenFinish builds txbuilder Reserve and Receiver implementations
// for a voting token finish/close transition.
func TokenFinish(ctx context.Context, token *Token) (txbuilder.Reserver, txbuilder.Receiver, error) {
	data := token.tokenScriptData
	data.State = data.State | stateFinished

	adminAddr, err := appdb.GetAddress(ctx, token.AdminScript)
	if err != nil {
		adminAddr = nil
	}

	reserver := tokenReserver{
		outpoint:   token.Outpoint,
		clause:     clauseFinish,
		output:     data,
		prevScript: token.tokenScriptData.PKScript(),
		adminAddr:  adminAddr,
	}
	return reserver, data, nil
}

// TokenReset builds txbuilder.Reserve and Receiver implementations
// to reset a voting token.
func TokenReset(ctx context.Context, token *Token, preserveRegistration bool, quorumSecretHash bc.Hash) (txbuilder.Reserver, txbuilder.Receiver, error) {
	data := tokenScriptData{
		Right:       token.Right,
		AdminScript: token.AdminScript,
		OptionCount: token.OptionCount,
		State:       stateDistributed,
		SecretHash:  quorumSecretHash,
		Vote:        0, // unset vote
	}
	if preserveRegistration && (token.State.Registered() || token.State.Voted()) {
		data.State = stateRegistered
	}

	adminAddr, err := appdb.GetAddress(ctx, token.AdminScript)
	if err != nil {
		adminAddr = nil
	}

	reserver := tokenReserver{
		outpoint:   token.Outpoint,
		clause:     clauseReset,
		output:     data,
		prevScript: token.tokenScriptData.PKScript(),
		adminAddr:  adminAddr,
	}
	return reserver, data, nil
}

// TokenRetire builds a txbuilder Reserver implementation for retiring a
// voting token.
func TokenRetire(ctx context.Context, token *Token) (txbuilder.Reserver, error) {
	adminAddr, err := appdb.GetAddress(ctx, token.AdminScript)
	if err != nil {
		adminAddr = nil
	}

	reserver := tokenReserver{
		outpoint:   token.Outpoint,
		clause:     clauseRetire,
		prevScript: token.tokenScriptData.PKScript(),
		adminAddr:  adminAddr,
	}
	return reserver, nil
}
