package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
)

// IssuerNode represents a single issuer ndoe. It is intended to be used wth API
// responses.
type IssuerNode struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Keys     []*NodeKey `json:"keys,omitempty"`
	SigsReqd int        `json:"signatures_required"`
}

// InsertIssuerNode adds the issuer node to the database
func InsertIssuerNode(ctx context.Context, projID, label string, xpubs []*hd25519.XPub, gennedKeys []*hd25519.XPrv, sigsRequired int, clientToken *string) (*IssuerNode, error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction

	const q = `
		INSERT INTO issuer_nodes (label, project_id, keyset, generated_keys, sigs_required, client_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (project_id, client_token) DO NOTHING
		RETURNING id
	`
	var id string
	err := pg.QueryRow(ctx, q,
		label,
		projID,
		pg.Strings(xpubsToStrings(xpubs)),
		pg.Strings(xprvsToStrings(gennedKeys)),
		sigsRequired,
		clientToken,
	).Scan(&id)
	if err == sql.ErrNoRows && clientToken != nil {
		// A sql.ErrNoRows error here indicates that we failed to insert
		// an issuer node because there was a conflict on the client token.
		// A previous request to create this issuer node succeeded.
		return nil, errors.Wrap(err, "looking up existing asset issuser")
	}
	if err != nil {
		return nil, errors.Wrap(err, "insert asset issuer")
	}

	keys, err := buildNodeKeys(xpubs, gennedKeys)
	if err != nil {
		return nil, errors.Wrap(err, "generating asset issuer key list")
	}

	return &IssuerNode{
		ID:       id,
		Label:    label,
		Keys:     keys,
		SigsReqd: sigsRequired,
	}, nil
}

// NextAsset returns all data needed
// for creating a new asset. This includes
// all keys, the issuer node index, a
// new index for the asset being created,
// and the number of signatures required.
func NextAsset(ctx context.Context, inodeID string) (asset *Asset, sigsRequired int, err error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		SELECT keyset,
		key_index(key_index),
		key_index(nextval('assets_key_index_seq'::regclass)-1),
		sigs_required FROM issuer_nodes
		WHERE id=$1
	`
	asset = &Asset{IssuerNodeID: inodeID}
	var (
		xpubs   []string
		sigsReq int
	)
	err = pg.QueryRow(ctx, q, inodeID).Scan(
		(*pg.Strings)(&xpubs),
		(*pg.Uint32s)(&asset.INIndex),
		(*pg.Uint32s)(&asset.AIndex),
		&sigsReq,
	)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, 0, errors.WithDetailf(err, "asset issuer %v: get key info", inodeID)
	}

	asset.Keys, err = stringsToXPubs(xpubs)
	if err != nil {
		return nil, 0, errors.Wrap(err, "parsing keys")
	}

	return asset, sigsReq, nil
}
