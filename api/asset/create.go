package asset

import (
	"bytes"
	"sort"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	chaintxscript "chain/fedchain/txscript"
)

// Create generates a new asset redeem script
// and id inside of an asset group.
func Create(ctx context.Context, agID, label string) (*appdb.Asset, error) {
	if label == "" {
		return nil, appdb.ErrBadLabel
	}

	asset, sigsReq, err := appdb.NextAsset(ctx, agID)
	if err != nil {
		return nil, errors.Wrap(err, "getting asset key info")
	}

	asset.Label = label

	var pubkeys []*btcutil.AddressPubKey
	for _, key := range asset.Keys {
		pubkeys = append(pubkeys, addrPubKey(key, assetIssuanceDerivationPath(asset)))
	}
	sort.Sort(pubKeysByAddress(pubkeys))

	asset.RedeemScript, err = txscript.MultiSigScript(pubkeys, sigsReq)
	if err != nil {
		return nil, errors.Wrapf(err, "creating asset: group id %v sigsReq %v", agID, sigsReq)
	}
	pkScript, err := chaintxscript.RedeemToPkScript(asset.RedeemScript)
	if err != nil {
		return nil, err
	}
	asset.Hash = chaintxscript.PkScriptToAssetID(pkScript)

	err = appdb.InsertAsset(ctx, asset)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset")
	}

	return asset, nil
}

type pubKeysByAddress []*btcutil.AddressPubKey

func (b pubKeysByAddress) Len() int      { return len(b) }
func (b pubKeysByAddress) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b pubKeysByAddress) Less(i, j int) bool {
	ai := b[i].ScriptAddress()
	aj := b[j].ScriptAddress()
	return bytes.Compare(ai, aj) < 0
}

// The only error returned has a uniformly distributed probability of 1/2^127
// We've decided to ignore this chance.
func addrPubKey(key *appdb.Key, path []uint32) *btcutil.AddressPubKey {
	xpub := &key.XPub.ExtendedKey
	for _, p := range path {
		xpub, _ = xpub.Child(p)
	}
	eckey, _ := xpub.ECPubKey()
	addr, _ := btcutil.NewAddressPubKey(eckey.SerializeCompressed(), &chaincfg.MainNetParams)
	return addr
}
