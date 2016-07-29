package asset

import (
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/crypto/ed25519/hd25519"
	"chain/errors"
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
	IssuerNode nodeType = iota
)

// CreateNodeReq is a user filled struct
// passed into CreateIssuerNode
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
// passed into CreateIssuerNode.
// It is deprecated in favor of CreateNodeReq.
type DeprecatedCreateNodeReq struct {
	Label       string
	XPubs       []string
	GenerateKey bool `json:"generate_key"`
}

// CreateNodeKeySpec describes a single key in a node's multi-sig configuration.
// It consists of a type, plus parameters depending on that type.
// For issuer nodes, only "service" type is valid.
// For service-type keys, either the XPub field is explicitly provided, or the
// Generate flag is set to true, in which case the xprv/xpub will be generated
// on the server side.
type CreateNodeKeySpec struct {
	Type string

	// Parameters for type "node"
	XPub     string
	Generate bool
}

// CreateNode is used to create issuer nodes
func CreateNode(ctx context.Context, node nodeType, projID string, req *CreateNodeReq) (interface{}, error) {
	if req.Label == "" {
		return nil, errors.WithDetail(appdb.ErrBadLabel, "missing/null value")
	}

	if req.SigsRequired < 1 {
		return nil, ErrBadSigsRequired
	}

	var (
		xpubs       []*hd25519.XPub
		gennedXprvs []*hd25519.XPrv
	)

	variableKeyCount := 0
	for i, k := range req.Keys {
		switch k.Type {
		case "node":
			// For backward compatibility, we allow "node" as an alias for "service".
			fallthrough
		case "service":
			if k.XPub != "" {
				xpub, err := hd25519.XPubFromString(k.XPub)
				if err != nil {
					return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: xpub is not valid", i)
				}
				xpubs = append(xpubs, xpub)
			} else if k.Generate {
				xprv, xpub, err := hd25519.NewXKeys(nil)
				if err != nil {
					return nil, err
				}
				xpubs = append(xpubs, xpub)
				gennedXprvs = append(gennedXprvs, xprv)
			} else {
				return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: service key must be generated, or an explicit xpub", i)
			}
		default:
			return nil, errors.WithDetailf(ErrBadKeySpec, "key %d: invalid type %s", i, k.Type)
		}
	}

	if len(xpubs)+variableKeyCount < req.SigsRequired {
		return nil, ErrTooFewKeys
	}

	// Do nothing with variable keys for Issuer Nodes since they can't have variable keys yet.
	return appdb.InsertIssuerNode(ctx, projID, req.Label, xpubs, gennedXprvs, req.SigsRequired, req.ClientToken)
}
