package fedchain

import (
	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/fedchain/bc"
)

func (fc *FC) ApplyTx(ctx context.Context, tx *bc.Tx) error {
	return fc.applyTx(ctx, tx)
}

func IsSignedByTrustedHost(block *bc.Block, trustedKeys []*btcec.PublicKey) bool {
	return isSignedByTrustedHost(block, trustedKeys)
}
