package asset

import (
	"bytes"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain/wire"
)

func Issue(ctx context.Context, assetID string, outs []Output) (*appdb.Tx, error) {
	tx := wire.NewMsgTx()
	tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}))

	asset, err := appdb.AssetByID(ctx, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "get asset by ID")
	}

	err = addAssetIssuanceOutputs(tx, asset, outs)
	if err != nil {
		return nil, errors.Wrap(err, "add issuance outputs")
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)
	appTx := &appdb.Tx{
		Unsigned:   buf.Bytes(),
		BlockChain: "sandbox", // TODO(tess): make this BlockChain: blockchain.FromContext(ctx)
		Inputs:     []*appdb.Input{asset.IssuanceInput()},
	}
	return appTx, nil
}

type Output struct {
	Address  string
	BucketID string
	Amount   int64
}

func addAssetIssuanceOutputs(tx *wire.MsgTx, asset *appdb.Asset, outs []Output) error {
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
