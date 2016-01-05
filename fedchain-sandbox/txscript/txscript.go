package txscript

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

// RedeemToPkScript takes a p2sh redeem script
// and returns the pkscript to pay to it.
func RedeemToPkScript(redeem []byte) []byte {
	p2sh, _ := btcutil.NewAddressScriptHash(redeem, &chaincfg.MainNetParams)
	script, _ := txscript.PayToAddrScript(p2sh)
	return script
}
