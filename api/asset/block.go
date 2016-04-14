package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/asset/nodetxlog"
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
		fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
			err := nodetxlog.Write(ctx, tx, time.Now())
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "writing activitiy"))
			}
		})
	}
}
