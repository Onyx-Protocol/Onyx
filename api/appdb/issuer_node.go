package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/hdkey"
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
func InsertIssuerNode(ctx context.Context, projID, label string, xpubs, gennedKeys []*hdkey.XKey, sigsRequired int, clientToken *string) (*IssuerNode, error) {
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
		pg.Strings(keysToStrings(xpubs)),
		pg.Strings(keysToStrings(gennedKeys)),
		sigsRequired,
		clientToken,
	).Scan(&id)
	if err == sql.ErrNoRows && clientToken != nil {
		// A sql.ErrNoRows error here indicates that we failed to insert
		// an issuer node because there was a conflict on the client token.
		// A previous request to create this issuer node succeeded.
		in, err := getIssuerNodeByClientToken(ctx, projID, *clientToken)
		return in, errors.Wrap(err, "looking up existing issuer node")
	}
	if err != nil {
		return nil, errors.Wrap(err, "insert issuer node")
	}

	keys, err := buildNodeKeys(xpubs, gennedKeys)
	if err != nil {
		return nil, errors.Wrap(err, "generating node key list")
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
		return nil, 0, errors.WithDetailf(err, "issuer node %v: get key info", inodeID)
	}

	asset.Keys, err = stringsToKeys(xpubs)
	if err != nil {
		return nil, 0, errors.Wrap(err, "parsing keys")
	}

	return asset, sigsReq, nil
}

// ListIssuerNodes returns a list of issuer nodes belonging to the given
// project.
func ListIssuerNodes(ctx context.Context, projID string) ([]*IssuerNode, error) {
	q := `
		SELECT id, label
		FROM issuer_nodes
		WHERE project_id = $1 AND NOT archived
		ORDER BY id
	`
	rows, err := pg.Query(ctx, q, projID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var inodes []*IssuerNode
	for rows.Next() {
		inode := new(IssuerNode)
		err := rows.Scan(&inode.ID, &inode.Label)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		inodes = append(inodes, inode)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return inodes, nil
}

// getIssuerNodeByClientToken returns basic information about a single issuer
// node, looking the node up by its project ID and its client token.
func getIssuerNodeByClientToken(ctx context.Context, projID, clientToken string) (*IssuerNode, error) {
	inq := issuerNodeQuery{
		projectID:   projID,
		clientToken: clientToken,
	}
	issuerNode, err := lookupIssuerNode(ctx, inq)
	return issuerNode, errors.WithDetailf(err, "project ID: %s, client token: %s", projID, clientToken)
}

// GetIssuerNode returns basic information about a single issuer node.
func GetIssuerNode(ctx context.Context, issuerNodeID string) (*IssuerNode, error) {
	issuerNode, err := lookupIssuerNode(ctx, issuerNodeQuery{id: issuerNodeID})
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	return issuerNode, errors.WithDetailf(err, "issuer node ID: %v", issuerNodeID)
}

type issuerNodeQuery struct {
	id          string
	projectID   string
	clientToken string
}

func lookupIssuerNode(ctx context.Context, inq issuerNodeQuery) (*IssuerNode, error) {
	const (
		baseQ = `
			SELECT id, label, keyset, generated_keys, sigs_required
			FROM issuer_nodes
		`
	)
	var (
		q         string
		queryArgs []interface{}
	)

	if inq.projectID != "" && inq.clientToken != "" {
		q = baseQ + "WHERE project_id = $1 AND client_token = $2"
		queryArgs = []interface{}{inq.projectID, inq.clientToken}
	} else {
		q = baseQ + "WHERE id = $1"
		queryArgs = []interface{}{inq.id}
	}

	var (
		id          string
		label       string
		pubKeyStrs  []string
		privKeyStrs []string
		sigsReqd    int
	)
	err := pg.QueryRow(ctx, q, queryArgs...).Scan(
		&id,
		&label,
		(*pg.Strings)(&pubKeyStrs),
		(*pg.Strings)(&privKeyStrs),
		&sigsReqd,
	)
	if err != nil {
		return nil, err
	}

	xpubs, err := stringsToKeys(pubKeyStrs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing pub keys")
	}

	xprvs, err := stringsToKeys(privKeyStrs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing private keys")
	}

	keys, err := buildNodeKeys(xpubs, xprvs)
	if err != nil {
		return nil, errors.Wrap(err, "generating node key list")
	}

	return &IssuerNode{
		ID:       id,
		Label:    label,
		Keys:     keys,
		SigsReqd: sigsReqd,
	}, nil
}

// UpdateIssuerNode updates the label of an issuer node.
func UpdateIssuerNode(ctx context.Context, inodeID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE issuer_nodes SET label = $2 WHERE id = $1`
	_, err := pg.Exec(ctx, q, inodeID, *label)
	return errors.Wrap(err, "update query")
}

// ArchiveIssuerNode marks an issuer node as archived.
// Archived issuer nodes do not appear for their parent projects,
// in the dashboard or for listIssuerNodes. They cannot create new
// assets, and their preexisting assets become archived.
//
// Must be called inside a database transaction.
func ArchiveIssuerNode(ctx context.Context, inodeID string) error {
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction

	const assetQ = `UPDATE assets SET archived = true WHERE issuer_node_id = $1`
	_, err := pg.Exec(ctx, assetQ, inodeID)
	if err != nil {
		return errors.Wrap(err, "archiving assets")
	}

	const q = `UPDATE issuer_nodes SET archived = true WHERE id = $1`
	_, err = pg.Exec(ctx, q, inodeID)
	if err != nil {
		return errors.Wrap(err, "archive query")
	}

	return nil
}
