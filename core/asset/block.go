package asset

import (
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
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
		fc.AddBlockCallback(func(ctx context.Context, b *bc.Block) {
			recordIssuances(ctx, b)
		})
	}
}
