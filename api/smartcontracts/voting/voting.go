package voting

import (
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/errors"
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

// TODO(jackson): Ensure that updateIndexes is idempotent. There might be
// tricky cases for recall and override.
func updateIndexes(ctx context.Context, blockHeight uint64, blockTxIndex int, tx *bc.Tx) error {
	// For inputs that redeem an existing voting rights contract, void any
	// voting right claims that we've indexed which were voided by a recall,
	// transfer or override.
	for _, in := range tx.Inputs {
		ok, clause, ownershipHash := testRightsSigscript(in.SignatureScript)
		if !ok {
			continue
		}

		var err error
		switch clause {
		case clauseAuthenticate:
			// Void the voting right claim at the previous outpoint.
			// The holder will need to use the new tx's outpoint from
			// now on.
			err = voidVotingRight(ctx, in.Previous)
		case clauseTransfer:
			// Void the voting right claim at the previous outpoint.
			// A transferred voting right cannot be recalled by the
			// transferer.
			err = voidVotingRight(ctx, in.Previous)
		case clauseRecall:
			// Void all of the voting right claims for this token, starting
			// at the recall point.
			err = voidRecalledVotingRights(ctx, in.Previous, ownershipHash)
		case clauseOverride:
			// TODO(jackson): The override clause will require us to void
			// previous voting right claims as well.
		}
		if err != nil {
			log.Error(ctx, err)
		}
	}

	// For outputs that match one of the voting contracts' p2c script
	// formats, index voting-specific info in the db.
	for i, out := range tx.Outputs {
		scriptData, err := testRightsContract(out.Script)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "testing for voting rights output script"))
			continue
		}
		if scriptData == nil {
			continue
		}

		err = insertVotingRight(ctx, out.AssetID, blockHeight, blockTxIndex, bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}, *scriptData)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "upserting voting rights"))
			continue
		}
	}
	return nil
}
