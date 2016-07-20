package voting

import (
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/log"
)

var fc *cos.FC

// Connect installs hooks to notify the voting package of new
// transactions. The voting package compares all transaction outputs
// to the voting contracts to update indexes appropriately.
func Connect(chain *cos.FC) {
	if fc == chain {
		// Silently ignore duplicate calls.
		return
	}
	fc = chain

	// Use a block callback instead of a tx callback so that we have access
	// to the transaction's position within the block. The position determines
	// ordering of transactions within the same block, and is used when
	// reconstructing ownership chains.
	//
	// TODO(jackson): Ensure crash recovery handles cos callbacks.
	fc.AddBlockCallback(func(ctx context.Context, block *bc.Block) {
		for i, tx := range block.Transactions {
			err := updateIndexes(ctx, block.Height, i, tx)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	})
}

func updateIndexes(ctx context.Context, blockHeight uint64, blockTxIndex int, tx *bc.Tx) (err error) {
	var (
		votingRightInputs    = map[bc.AssetID]bc.TxInput{}
		votingRightOutputs   = map[bc.AssetID]rightScriptData{}
		votingRightOutpoints = map[bc.AssetID]bc.Outpoint{}
	)

	for _, in := range tx.Inputs {
		// Collect all of the voting right inputs into the maps.
		if ok, _, _ := testRightsSigscript(in.SignatureScript); ok {
			votingRightInputs[in.AssetAmount.AssetID] = *in
		}

		// Delete any voting tokens that are consumed.
		if ok, _, _ := testTokensSigscript(in.SignatureScript); ok {
			err := voidVotingTokens(ctx, in.Previous)
			if err != nil {
				return err
			}
		}
	}

	for i, out := range tx.Outputs {
		outpoint := bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}

		// Collect all of the voting right outputs.
		if rightData, err := testRightsContract(out.ControlProgram); rightData != nil && err == nil {
			votingRightOutputs[out.AssetID] = *rightData
			votingRightOutpoints[out.AssetID] = outpoint
			continue
		}

		// If the output is a voting token, update the voting token index.
		if tokenData, err := testTokenContract(out.ControlProgram); tokenData != nil && err == nil {
			err = insertVotingToken(ctx, out.AssetID, blockHeight, outpoint, out.Amount, *tokenData)
			if err != nil {
				return err
			}
		}
	}

	// Index all of the changes to voting rights.
	for assetID, data := range votingRightOutputs {
		// If there is no voting right input, then this is a voting right issuance.
		if _, ok := votingRightInputs[assetID]; !ok {
			err = insertVotingRight(ctx, assetID, 0, blockHeight, votingRightOutpoints[assetID], data)
			if err != nil {
				return err
			}
			continue
		}
		in := votingRightInputs[assetID]
		_, clause, params := testRightsSigscript(in.SignatureScript)

		// Look up the current state of the voting right by finding the voting
		// right at the previous outpoint with the highest ordinal.
		prev, err := FindRightPrevout(ctx, assetID, in.Previous)
		if err != nil {
			return err
		}

		switch clause {
		case clauseAuthenticate:
			// Void the voting right claim at the previous outpoint.
			// The holder will need to use the new tx's outpoint from now on.
			err = voidVotingRights(ctx, assetID, blockHeight, prev.Ordinal, prev.Ordinal)
			if err != nil {
				return err
			}
			err = insertVotingRight(ctx, assetID, prev.Ordinal+1, blockHeight, votingRightOutpoints[assetID], data)
			if err != nil {
				return err
			}
		case clauseTransfer:
			// Void the voting right claim at the previous outpoint.
			// A transferred voting right cannot be recalled by the transferer
			err = voidVotingRights(ctx, assetID, blockHeight, prev.Ordinal, prev.Ordinal)
			if err != nil {
				return err
			}
			err = insertVotingRight(ctx, assetID, prev.Ordinal+1, blockHeight, votingRightOutpoints[assetID], data)
			if err != nil {
				return err
			}
		case clauseDelegate:
			// Nothing to void, just increment the ordinal on the new right.
			err = insertVotingRight(ctx, assetID, prev.Ordinal+1, blockHeight, votingRightOutpoints[assetID], data)
			if err != nil {
				return err
			}
		case clauseRecall:
			// Use the ownership chain to find the recall point.
			var recallChain bc.Hash
			ok := true
			params, recallChain = paramsPopHash(params, &ok)
			if !ok {
				continue
			}

			recallPoint, err := findRecallPoint(ctx, assetID, prev.Ordinal, recallChain)
			if err != nil {
				return err
			}
			err = voidVotingRights(ctx, assetID, blockHeight, recallPoint.Ordinal, prev.Ordinal)
			if err != nil {
				return err
			}
			err = insertVotingRight(ctx, assetID, prev.Ordinal+1, blockHeight, votingRightOutpoints[assetID], data)
			if err != nil {
				return err
			}
		case clauseOverride:
			// Pull all of the override data out of the sigscript parameters.
			valid := true
			params, delegatable := paramsPopBool(params, &valid)
			params, forkHash := paramsPopHash(params, &valid)

			// Pop off and discard all of the proof hashes. We don't need
			// them for indexing.
			params, proofHashCount := paramsPopInt64(params, &valid)
			for i := int64(0); i < proofHashCount; i++ {
				params, _ = paramsPopHash(params, &valid)
			}

			params, newHolderCount := paramsPopInt64(params, &valid)
			newHolders := make([]RightHolder, newHolderCount)
			for i := range newHolders {
				params, newHolders[i].Script = paramsPopBytes(params, &valid)
			}

			// If any one of the sigscript parameters failed to decode, this
			// sigscript doesn't match.
			if !valid {
				continue
			}

			// Void all of the voting rights from [forkHash, ..., ..., prevOut] inclusive.
			recallPoint, err := findRecallPoint(ctx, assetID, prev.Ordinal, forkHash)
			if err != nil {
				return err
			}
			err = voidVotingRights(ctx, assetID, blockHeight, recallPoint.Ordinal, prev.Ordinal)
			if err != nil {
				return err
			}

			// Insert all of the new voting right holders.
			prevData := recallPoint.rightScriptData
			prevData.Delegatable = delegatable
			nextOrdinal := prev.Ordinal + 1
			for _, nh := range newHolders {
				prevData.HolderScript = nh.Script

				err = insertVotingRight(ctx, assetID, nextOrdinal, blockHeight, votingRightOutpoints[assetID], prevData)
				if err != nil {
					return err
				}
				nextOrdinal++
				prevData.OwnershipChain = calculateOwnershipChain(prevData.OwnershipChain, prevData.HolderScript)
			}
		}
	}
	return nil
}
