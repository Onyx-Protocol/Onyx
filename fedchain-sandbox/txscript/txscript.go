package txscript

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"

	"chain/errors"
)

// AddrPkScript takes a base58-encoded address
// and generates a PkScript for use on a TxOut.
func AddrPkScript(addr string) ([]byte, error) {
	address, err := btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, err
	}
	return pkScript, nil
}

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

// RedeemToPkScript takes a p2sh redeem script
// and returns the pkscript to pay to it.
func RedeemToPkScript(redeem []byte) []byte {
	p2sh, _ := btcutil.NewAddressScriptHash(redeem, &chaincfg.MainNetParams)
	script, _ := txscript.PayToAddrScript(p2sh)
	return script
}
