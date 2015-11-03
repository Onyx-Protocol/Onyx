package txscript

import (
	"chain/errors"
	"chain/fedchain/script"

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

// RedeemScriptFromP2SHSigScript parses the signature script and returns the
// redeem script.
func RedeemScriptFromP2SHSigScript(sigScript script.Script) ([]byte, error) {
	opCodes, err := parseScript(sigScript)
	if err != nil {
		return nil, errors.Wrap(err, "decoding redeem script from sig script")
	}

	return opCodes[len(opCodes)-1].data, nil
}

// TODO(tessr): Write BuildP2SHSigScript, which will correlate to
// RedeemScriptFromP2SHSigScript and will do something similar to
// asset.assembleSignatures.
