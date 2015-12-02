package asset

import (
	"github.com/btcsuite/btcutil/hdkeychain"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

var (
	// ErrBadXPub is returned by CreateNode.
	// It may be wrapped using package chain/errors.
	ErrBadXPub = errors.New("bad xpub")

	// ErrTooFewKeys can be returned by CreateNode if not enough keys
	// have been provided or generated.
	ErrTooFewKeys = errors.New("too few keys for signatures required")
)

type nodeType int

// Node types used for CreateNode
const (
	ManagerNode nodeType = iota
	IssuerNode  nodeType = iota
)

// CreateNodeReq is a user filled struct
// passed into CreateManagerNode or CreateIssuerNode
type CreateNodeReq struct {
	Label        string
	Keys         []*XPubInit
	SigsRequired int `json:"signatures_required"`
}

// DeprecatedCreateNodeReq is a user filled struct
// passed into CreateManagerNode or CreateIssuerNode.
// It is deprecated in favor of CreateNodeReq.
type DeprecatedCreateNodeReq struct {
	Label       string
	XPubs       []string
	GenerateKey bool `json:"generate_key"`
}

// XPubInit is a representation of an xpub used when nodes are being created.
// It includes the key itself, as well as two flags:
// Generate specifies whether the key needs to be generated server-side, and
// Variable specifies whether this is a placeholder for an account-specific key.
// If Variable is true, Generate must be false and Key must be empty.
type XPubInit struct {
	Key      string
	Generate bool
	Variable bool
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

	variableKeyCount := 0
	for i, xpub := range req.Keys {
		if xpub.Generate {
			pub, priv, err := newKey()
			if err != nil {
				return nil, err
			}
			keys = append(keys, pub)
			gennedKeys = append(gennedKeys, priv)
		} else if xpub.Variable {
			variableKeyCount++
		} else {
			key, err := hdkey.NewXKey(xpub.Key)
			if err != nil {
				err = errors.Wrap(ErrBadXPub, err.Error())
				return nil, errors.WithDetailf(err, "xpub %d", i)
			}
			keys = append(keys, key)
		}
	}

	if len(keys)+variableKeyCount < req.SigsRequired {
		return nil, ErrTooFewKeys
	}

	for i, key := range keys {
		if key.IsPrivate() {
			return nil, errors.WithDetailf(ErrBadXPub, "key %d is xpriv, not xpub", i)
		}
	}

	if node == ManagerNode {
		return appdb.InsertManagerNode(ctx, projID, req.Label, keys, gennedKeys, variableKeyCount, req.SigsRequired)
	}
	// Do nothing with variable keys for Issuer Nodes since they can't have variable keys yet.
	return appdb.InsertIssuerNode(ctx, projID, req.Label, keys, gennedKeys, req.SigsRequired)
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
	return &hdkey.XKey{ExtendedKey: *xpub}, &hdkey.XKey{ExtendedKey: *xprv}, nil
}
