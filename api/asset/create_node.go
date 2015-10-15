package asset

import (
	"github.com/btcsuite/btcutil/hdkeychain"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

// Errors returned by CreateWallet.
// May be wrapped using package chain/errors.
var (
	ErrBadXPubCount = errors.New("bad xpub count")
	ErrBadXPub      = errors.New("bad xpub")
)

type nodeType int

// Node types used for CreateNode
const (
	ManagerNode nodeType = iota
	IssuerNode  nodeType = iota
)

// CreateNodeReq is a user filled struct
// passed into CreateWallet or CreateAssetGroup
type CreateNodeReq struct {
	Label       string
	XPubs       []string
	GenerateKey bool `json:"generate_key"`
}

// CreateNode is used to create manager and issuer nodes
func CreateNode(ctx context.Context, node nodeType, projID string, req *CreateNodeReq) (interface{}, error) {
	if req.Label == "" {
		return nil, appdb.ErrBadLabel
	}

	var (
		keys       []*hdkey.XKey
		gennedKeys []*hdkey.XKey
	)
	for i, xpub := range req.XPubs {
		key, err := hdkey.NewXKey(xpub)
		if err != nil {
			err = errors.Wrap(ErrBadXPub, err.Error())
			return nil, errors.WithDetailf(err, "xpub %d", i)
		}
		keys = append(keys, key)
	}

	if req.GenerateKey {
		pub, priv, err := newKey()
		if err != nil {
			return nil, err
		}
		keys = append(keys, pub)
		gennedKeys = append(gennedKeys, priv)
	}

	if len(keys) != 1 {
		// only 1-of-1 supported so far
		return nil, ErrBadXPubCount
	}
	for i, key := range keys {
		if key.IsPrivate() {
			err := errors.WithDetailf(ErrBadXPub, "key number %d", i)
			return nil, errors.WithDetail(err, "key is xpriv, not xpub")
		}
	}

	if node == ManagerNode {
		return appdb.InsertWallet(ctx, projID, req.Label, keys, gennedKeys)
	}
	return appdb.InsertAssetGroup(ctx, projID, req.Label, keys, gennedKeys)
}

func newKey() (pub, priv *hdkey.XKey, err error) {
	seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating key seed")
	}
	xprv, err := hdkeychain.NewMaster(seed)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating root xprv")
	}
	xpub, err := xprv.Neuter()
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting root xpub")
	}
	return &hdkey.XKey{*xpub}, &hdkey.XKey{*xprv}, nil
}
