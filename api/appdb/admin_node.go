package appdb

import (
	"chain/database/pg"
	"chain/errors"
	"database/sql"

	"golang.org/x/net/context"
)

// Everything in this file is DEPRECATED. Modeling multiple admin nodes is not
// part of our immediate product roadmap.

var ErrAdminNodeAlreadyExists = errors.New("An admin node for this blockchain already exists.")

// AdminNode represents a single admin node. It is intended to be used wth API
// responses.
type AdminNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// InsertAdminNode inserts a new admin node into the database.
func InsertAdminNode(ctx context.Context, projID, label string) (*AdminNode, error) {
	if label == "" {
		return nil, ErrBadLabel
	}

	// Currently, we only allow one admin node per blockchain (i.e., per
	// database.)
	const q = `
		INSERT INTO admin_nodes (label, project_id)
		SELECT $1, $2 WHERE NOT EXISTS (SELECT * FROM admin_nodes)
		RETURNING id
	`
	var id string
	err := pg.FromContext(ctx).QueryRow(q, label, projID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, ErrAdminNodeAlreadyExists
	}
	if err != nil {
		return nil, errors.Wrap(err, "insert admin node")
	}

	return &AdminNode{
		ID:    id,
		Label: label,
	}, nil
}

// GetAdminNode returns basic information about a single admin node.
func GetAdminNode(ctx context.Context, adminNodeID string) (*AdminNode, error) {
	var (
		q = `
			SELECT label
			FROM admin_nodes
			WHERE id = $1
		`
		label string
	)
	err := pg.FromContext(ctx).QueryRow(q, adminNodeID).Scan(&label)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "admin node ID: %v", adminNodeID)
	}
	if err != nil {
		return nil, err
	}

	return &AdminNode{
		ID:    adminNodeID,
		Label: label,
	}, nil
}

// ListAdminNodes returns a list of admin nodes contained in the given project.
func ListAdminNodes(ctx context.Context, projID string) ([]*AdminNode, error) {
	q := `
		SELECT id, label
		FROM admin_nodes
		WHERE project_id = $1
		ORDER BY created_at
	`
	rows, err := pg.FromContext(ctx).Query(q, projID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var adminNodes []*AdminNode
	for rows.Next() {
		m := new(AdminNode)
		err := rows.Scan(&m.ID, &m.Label)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		adminNodes = append(adminNodes, m)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return adminNodes, nil
}

// UpdateAdminNode updates the label of an admin node.
func UpdateAdminNode(ctx context.Context, anodeID string, label *string) error {
	if label == nil {
		return nil
	}
	if *label == "" {
		return ErrBadLabel
	}

	const q = `UPDATE admin_nodes SET label = $2 WHERE id = $1`
	db := pg.FromContext(ctx)
	_, err := db.Exec(q, anodeID, *label)
	return errors.Wrap(err, "update query")
}

// DeleteAdminNode deletes the admin node.
func DeleteAdminNode(ctx context.Context, anodeID string) error {
	const q = `DELETE FROM admin_nodes WHERE id = $1`
	db := pg.FromContext(ctx)
	result, err := db.Exec(q, anodeID)
	if err != nil {
		return errors.Wrap(err, "delete query")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected")
	}
	if rowsAffected == 0 {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "admin node ID %v", anodeID)
	}
	return nil
}
