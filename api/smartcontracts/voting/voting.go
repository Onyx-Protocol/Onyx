package voting

import (
	"golang.org/x/net/context"

	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/log"
)

var fc *fedchain.FC

// ConnectFedchain installs hooks to notify the voting package of new
// transactions. The voting package compares all transaction outputs
// to the voting contracts to update indexes appropriately.
func ConnectFedchain(chain *fedchain.FC) {
	if fc == chain {
		// Silently ignore duplicate calls.
		return
	}
	fc = chain

	fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
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

			err = insertVotingRight(ctx, out.AssetID, bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}, *scriptData)
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "upserting voting rights"))
				continue
			}
		}
	})
}
