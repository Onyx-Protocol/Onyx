package main

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"

	"chain/fedchain/wire"
	"chain/wallets"
)

type output struct {
	Address  string
	BucketID string
	Amount   int64
}

func addAssetIssuanceOutputs(tx *wire.MsgTx, asset *wallets.Asset, outs []output) error {
	for _, out := range outs {
		if out.BucketID != "" {
			// TODO(erykwalder): actually generate a receiver
			// This address doesn't mean anything, it was grabbed from the internet.
			// We don't have its private key.
			out.Address = "1ByEd6DMfTERyT4JsVSLDoUcLpJTD93ifq"
		}

		addr, err := btcutil.DecodeAddress(out.Address, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return err
		}

		tx.AddTxOut(wire.NewTxOut(asset.Hash, out.Amount, pkScript))
	}
	return nil
}
