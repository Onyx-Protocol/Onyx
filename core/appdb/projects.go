package appdb

import (
	"database/sql"
	"sort"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/strings"
)

// Project represents a project. It can be used safely for API
// responses.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Errors returned from project- and user-related functions.
var (
	ErrBadProjectName = errors.New("invalid project name")
	ErrBadRole        = errors.New("invalid role")
)

// CreateProject creates a new project.
//
// Must be called inside a database transaction.
func CreateProject(ctx context.Context, name string) (*Project, error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction

	if name == "" {
		return nil, errors.WithDetail(ErrBadProjectName, "missing/null value")
	}

	var (
		q  = `INSERT INTO projects (name) VALUES ($1) RETURNING id`
		id string
	)
	err := pg.QueryRow(ctx, q, name).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert query")
	}

	return &Project{ID: id, Name: name}, nil
}

// ListProjects returns a list of active projects.
func ListProjects(ctx context.Context) ([]*Project, error) {
	q := `
		SELECT id, name
		FROM projects
		WHERE NOT archived
		ORDER BY name
	`
	rows, err := pg.Query(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := new(Project)
		err := rows.Scan(&p.ID, &p.Name)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return projects, nil
}

// GetProject returns information about a single project.
func GetProject(ctx context.Context, projID string) (*Project, error) {
	var (
		q    = `SELECT name FROM projects WHERE id = $1`
		name string
	)
	err := pg.QueryRow(ctx, q, projID).Scan(&name)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "project ID: %v", projID)
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}

	return &Project{ID: projID, Name: name}, nil
}

// UpdateProject updates project properties. If the project does not
// exist, an error with pg.ErrUserInputNotFound as the root is returned.
func UpdateProject(ctx context.Context, projID, name string) error {
	q := `UPDATE projects SET name = $1 WHERE id = $2 RETURNING 1`
	err := pg.QueryRow(ctx, q, name, projID).Scan(new(int))
	if err == sql.ErrNoRows {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "project ID: %v", projID)
	}
	if err != nil {
		return errors.Wrap(err, "update query")
	}
	return nil
}

// ArchiveProject marks a project as archived, hiding it from listProjects and
// archiving all of its issuers and assets.
//
// Must be called inside a database transaction.
func ArchiveProject(ctx context.Context, projID string) error {
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction

	const q = `UPDATE projects SET archived = true WHERE id = $1 RETURNING 1`
	err := pg.QueryRow(ctx, q, projID).Scan(new(int))
	if err == sql.ErrNoRows {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "project ID: %v", projID)
	}
	if err != nil {
		return errors.Wrap(err, "archive query")
	}

	const inQ = `UPDATE issuer_nodes SET archived = true WHERE project_id = $1`
	if _, err := pg.Exec(ctx, inQ, projID); err != nil {
		return errors.Wrap(err, "archive asset issuers query")
	}

	const assetQ = `
		UPDATE assets SET archived = true WHERE issuer_node_id IN (
			SELECT id FROM issuer_nodes WHERE project_id = $1
		)
	`
	if _, err := pg.Exec(ctx, assetQ, projID); err != nil {
		return errors.Wrap(err, "archive assets query")
	}

	return nil
}

// IsAdmin returns true if the user is an admin.
func IsAdmin(ctx context.Context, userID string) (bool, error) {
	const q = `
		SELECT COUNT(*)=1 FROM users
		WHERE id=$1 AND role='admin'
	`
	var isAdmin bool
	row := pg.QueryRow(ctx, q, userID)
	err := row.Scan(&isAdmin)
	return isAdmin, errors.Wrap(err)
}

// CheckActiveIssuer returns nil if the provided asset issuer is active.
// If the asset issuer has been archived, this function returns ErrArchived.
func CheckActiveIssuer(ctx context.Context, issuerID string) error {
	const q = `
		SELECT archived
		FROM issuer_nodes WHERE id=$1
	`
	var archived bool
	err := pg.QueryRow(ctx, q, issuerID).Scan(&archived)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if archived {
		err = ErrArchived
	}
	return errors.WithDetailf(err, "issuer node ID: %v", issuerID)
}

// CheckActiveAsset returns nil if the provided assets are active.
// If any of the assets are archived, this function returns ErrArchived.
func CheckActiveAsset(ctx context.Context, assetIDs ...string) error {
	// Remove duplicates so that we know how many assets to expect.
	sort.Strings(assetIDs)
	assetIDs = strings.Uniq(assetIDs)

	const q = `
		SELECT COUNT(id),
		       COUNT(CASE WHEN archived THEN 1 ELSE NULL END) AS archived
		FROM assets
		WHERE id=ANY($1)
	`
	var (
		assetsArchived int
		assetsFound    int
	)
	err := pg.QueryRow(ctx, q, pg.Strings(assetIDs)).
		Scan(&assetsFound, &assetsArchived)
	if err != nil {
		return errors.Wrap(err)
	}
	if assetsFound != len(assetIDs) {
		err = pg.ErrUserInputNotFound
	} else if assetsArchived > 0 {
		err = ErrArchived
	}
	return errors.WithDetailf(err, "asset IDs: %+v", assetIDs)
}
