package asset

import (
	"github.com/btcsuite/btcutil/hdkeychain"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain/hdkey"
)

// Errors returned by functions in this file.
var (
	ErrBadSigsRequired = errors.New("zero signatures required")
	ErrBadKeySpec      = errors.New("bad key specification")
	ErrTooFewKeys      = errors.New("too few keys for signatures required")
)

type nodeType int

// Node types used for CreateNode
const (
	ManagerNode nodeType = iota
	IssuerNode  nodeType = iota
)

// CreateNodeReq is a user filled struct
// passed into CreateManagerNode or CreateIssuerNode
// TODO(jackson): ClientToken should become required once
// all the SDKs have been updated.
type CreateNodeReq struct {
	Label        string
	Keys         []*CreateNodeKeySpec
	SigsRequired int `json:"signatures_required"`

	// ClientToken is the application's unique token for the node. Every node
	// within a project should have a unique client token. The client token
	// is used to ensure idempotency of create node requests. Duplicate create
	// node requests within the same project with the same client_token will
	// only create one node.
	ClientToken *string `json:"client_token"`
}

// DeprecatedCreateNodeReq is a user filled struct
// passed into CreateManagerNode or CreateIssuerNode.
// It is deprecated in favor of CreateNodeReq.
type DeprecatedCreateNodeReq struct {
	Label       string
	XPubs       []string
	GenerateKey bool `json:"generate_key"`
}

// CreateNodeKeySpec describes a single key in a node's multi-sig configuration.
// It consists of a type, plus parameters depending on that type.
// Valid manager node types include "node" and "account". For issuer nodes,
// only "node" is valid.
// For node-type keys, either the XPub field is explicitly provided, or the
// Generate flag is set to true, in which case the xprv/xpub will be generated
// on the server side.
type CreateNodeKeySpec struct {
	Type string

	// Parameters for type "node"
	XPub     string
	Generate bool
}

// CreateNode is used to create manager and issuer nodes
func CreateNode(ctx context.Context, node nodeType, projID string, req *CreateNodeReq) (interface{}, error) {
	if req.Label == "" {
		return nil, appdb.ErrBadLabel
	}

	if req.SigsRequired < 1 {
		return nil, ErrBadSigsRequired
	}

	var (
		xpubs       []*hdkey.XKey
		gennedXprvs []*hdkey.XKey
	)

	variableKeyCount := 0
	for i, k := range req.Keys {
		switch k.Type {
		case "node":
			if k.XPub != "" {
				xpub, err := hdkey.NewXKey(k.XPub)
				if err != nil {
					return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: xpub is not valid", i)
				} else if xpub.IsPrivate() {
					return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: is xpriv, not xpub", i)
				}
				xpubs = append(xpubs, xpub)
			} else if k.Generate {
				xpub, xprv, err := newKey()
				if err != nil {
					return nil, err
				}
				xpubs = append(xpubs, xpub)
				gennedXprvs = append(gennedXprvs, xprv)
			} else {
				return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: node key must be generated, or an explicit xpub", i)
			}
		case "account":
			if node != ManagerNode {
				return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: account keys are only valid for manager nodes", i)
			}
			variableKeyCount++
		default:
			return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: invalid type %s", i, k.Type)
		}
	}

	if len(xpubs)+variableKeyCount < req.SigsRequired {
		return nil, ErrTooFewKeys
	}

	if node == ManagerNode {
		return appdb.InsertManagerNode(ctx, projID, req.Label, xpubs, gennedXprvs, variableKeyCount, req.SigsRequired, req.ClientToken)
	}

	// Do nothing with variable keys for Issuer Nodes since they can't have variable keys yet.
	return appdb.InsertIssuerNode(ctx, projID, req.Label, xpubs, gennedXprvs, req.SigsRequired, req.ClientToken)
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
