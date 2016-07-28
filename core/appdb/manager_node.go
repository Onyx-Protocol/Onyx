package appdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
)

// NodeKey is represents a single key in a node's multi-sig configuration.
// It is used as a return value when nodes are created.
//
// A NodeKey consists of a type, plus parameters depending on that type. Valid
// manager node types include "node" and "account". For issuer nodes, only
// "node" is valid.
//
// For node-type keys, XPrv will be populated only if it was generated
// server-side when the node was created.
type NodeKey struct {
	Type string `json:"type"`

	// Parameters for type "node"
	XPub *hd25519.XPub `json:"xpub,omitempty"`
	XPrv *hd25519.XPrv `json:"xprv,omitempty"`
}

func buildNodeKeys(xpubs []*hd25519.XPub, xprvs []*hd25519.XPrv) ([]*NodeKey, error) {
	pubToPrv := make(map[string]*hd25519.XPrv)
	for _, xprv := range xprvs {
		xpub := xprv.Public()
		pubToPrv[xpub.String()] = xprv
	}

	var res []*NodeKey
	for _, xpub := range xpubs {
		k := &NodeKey{Type: "service", XPub: xpub}

		s := xpub.String()
		if xprv := pubToPrv[s]; xprv != nil {
			k.XPrv = xprv
		}

		res = append(res, k)
	}

	return res, nil
}

// ManagerNode represents a single manager node. It is intended to be used wth API
// responses.
type ManagerNode struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Keys     []*NodeKey `json:"keys"`
	SigsReqd int        `json:"signatures_required"`
}

// InsertManagerNode inserts a new manager node into the database. If a manager node
// already exists with the provided project ID and client token, this function will
// return the existing manager node.
func InsertManagerNode(ctx context.Context, projID, label string, xpubs []*hd25519.XPub, gennedKeys []*hd25519.XPrv, variableKeys, sigsRequired int, clientToken *string) (w *ManagerNode, err error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	const q = `
		INSERT INTO manager_nodes (label, project_id, generated_keys, variable_keys, sigs_required, client_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (project_id, client_token) DO NOTHING
		RETURNING id
	`
	var id string
	xprvs := xprvsToStrings(gennedKeys)
	err = pg.QueryRow(ctx, q, label, projID, pg.Strings(xprvs), variableKeys, sigsRequired, clientToken).Scan(&id)
	if err == sql.ErrNoRows && clientToken != nil {
		// A sql.ErrNoRows error here indicates that we failed to insert
		// a manager node because there was a conflict on the client token.
		// A previous request to create this manager node succeeded.
		return nil, errors.Wrap(err, "looking up existing account manager")
	}
	if err != nil {
		return nil, errors.Wrap(err, "insert account manager")
	}

	err = createRotation(ctx, id, xpubsToStrings(xpubs)...)
	if err != nil {
		return nil, errors.Wrap(err, "create rotation")
	}

	keys, err := buildNodeKeys(xpubs, gennedKeys)
	if err != nil {
		return nil, errors.Wrap(err, "generating account manager key list")
	}

	for i := 0; i < variableKeys; i++ {
		keys = append(keys, &NodeKey{Type: "account"})
	}

	return &ManagerNode{
		ID:       id,
		Label:    label,
		Keys:     keys,
		SigsReqd: sigsRequired,
	}, nil
}

func createRotation(ctx context.Context, managerNodeID string, xpubs ...string) error {
	const q = `
		WITH new_rotation AS (
			INSERT INTO rotations (manager_node_id, keyset)
			VALUES ($1, $2)
			RETURNING id
		)
		UPDATE manager_nodes SET current_rotation=(SELECT id FROM new_rotation)
		WHERE id=$1
	`
	_, err := pg.Exec(ctx, q, managerNodeID, pg.Strings(xpubs))
	return err
}

func managerNodeVariableKeys(ctx context.Context, managerNodeID string) (int, error) {
	const q = `SELECT variable_keys FROM manager_nodes WHERE id = $1`
	count := 0
	err := pg.QueryRow(ctx, q, managerNodeID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
