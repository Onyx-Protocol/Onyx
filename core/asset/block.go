package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/asset/nodetxlog"
	"chain/cos"
	"chain/cos/bc"
	"chain/errors"
	"chain/log"
)

var fc *cos.FC

// Init sets the package level cos. If isManager is true,
// Init registers all necessary callbacks for updating
// application state with the cos.
func Init(chain *cos.FC, isManager bool) {
	if fc == chain {
		// Silently ignore duplicate calls.
		return
	}

	fc = chain
	if isManager {
		fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
			err := addAccountData(ctx, tx)
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "adding account data"))
			}
		})
		fc.AddBlockCallback(func(ctx context.Context, b *bc.Block, conflicts []*bc.Tx) {
			indexAccountUTXOs(ctx, b, conflicts)
			saveAssetDefinitions(ctx, b)
			recordIssuances(ctx, b, conflicts)
			for _, tx := range b.Transactions {
				// TODO(jackson): Once block timestamps are correctly populated
				// in milliseconds, Write() should use b.Time().
				err := nodetxlog.Write(ctx, tx, time.Now())
				if err != nil {
					log.Error(ctx, errors.Wrap(err, "writing activity"))
				}
			}
		})
	}
}
