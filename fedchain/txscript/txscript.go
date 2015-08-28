package txscript

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

// PkScriptAddr returns the address for a public key script,
// which is stored on a TxOut.
// It currently only supports p2sh addresses.
func PkScriptAddr(pkScript []byte) (btcutil.Address, error) {
	pushed, err := txscript.PushedData(pkScript)
	if err != nil {
		return nil, err
	}
	if len(pushed) != 1 || len(pushed[0]) != 20 {
		return nil, errors.New("output address is not p2sh")
	}
	addr, err := btcutil.NewAddressScriptHashFromHash(pushed[0], &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	return addr, nil
}
