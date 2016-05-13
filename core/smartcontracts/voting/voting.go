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
	fc.AddBlockCallback(func(ctx context.Context, block *bc.Block, conflicts []*bc.Tx) {
		for i, tx := range block.Transactions {
			err := updateIndexes(ctx, block.Height, i, tx)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	})
}

func updateIndexes(ctx context.Context, blockHeight uint64, blockTxIndex int, tx *bc.Tx) error {

	// We iterate through all the inputs, finding any sigscripts that redeem
	// an existing voting right contract. For each input, we void any previous
	// voting right owners that are no longer applicable. We also save the
	// ordinal of the consumed voting right output so that we know what ordinal
	// to use when inserting the new output.
	votingRightOrdinals := map[bc.AssetID]int{}
	for _, in := range tx.Inputs {
		ok, clause, params := testRightsSigscript(in.SignatureScript)
		if !ok {
			continue
		}

		// Look up the current state of the voting right by finding the voting
		// right at the previous outpoint with the highest ordinal.
		prev, err := FindRightPrevout(ctx, in.AssetAmount.AssetID, in.Previous)
		if err != nil {
			return err
		}

		switch clause {
		case clauseAuthenticate:
			// Void the voting right claim at the previous outpoint.
			// The holder will need to use the new tx's outpoint from now on.
			err = voidVotingRights(ctx, prev.AssetID, blockHeight, prev.Ordinal, prev.Ordinal)
			if err != nil {
				return err
			}
			votingRightOrdinals[prev.AssetID] = prev.Ordinal + 1
		case clauseTransfer:
			// Void the voting right claim at the previous outpoint.
			// A transferred voting right cannot be recalled by the transferer
			err = voidVotingRights(ctx, prev.AssetID, blockHeight, prev.Ordinal, prev.Ordinal)
			if err != nil {
				return err
			}
			votingRightOrdinals[prev.AssetID] = prev.Ordinal + 1
		case clauseDelegate:
			// Nothing to void, just increment the ordinal.
			votingRightOrdinals[prev.AssetID] = prev.Ordinal + 1
		case clauseRecall:
			// Use the ownership chain to find the recall point.
			var recallChain bc.Hash
			params, recallChain, ok = paramsPopHash(params)
			if !ok {
				continue
			}

			recallOrdinal, err := findRecallOrdinal(ctx, prev.AssetID, prev.Ordinal, recallChain)
			if err != nil {
				return err
			}

			err = voidVotingRights(ctx, prev.AssetID, blockHeight, recallOrdinal, prev.Ordinal)
			if err != nil {
				return err
			}
			votingRightOrdinals[prev.AssetID] = prev.Ordinal + 1
		case clauseOverride:
			// TODO(jackson): The override clause will require us to void
			// previous voting right claims as well.
		}
	}

	// For outputs that match one of the voting contracts' p2c script
	// formats, index voting-specific info in the db.
	for i, out := range tx.Outputs {
		outpoint := bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}

		// Newly issued voting rights will be indexed here.
		rightData, err := testRightsContract(out.Script)
		if err != nil {
			return err
		}
		if rightData != nil {
			ordinal := votingRightOrdinals[out.AssetID]
			err = insertVotingRight(ctx, out.AssetID, ordinal, blockHeight, outpoint, *rightData)
			if err != nil {
				return err
			}
			continue
		}

		// If the output is a voting token, update the voting token index.
		tokenData, err := testTokenContract(out.Script)
		if err != nil {
			return err
		}
		if tokenData != nil {
			err = upsertVotingToken(ctx, out.AssetID, blockHeight, outpoint, out.Amount, *tokenData)
			if err != nil {
				return err
			}
			continue
		}
	}
	return nil
}
