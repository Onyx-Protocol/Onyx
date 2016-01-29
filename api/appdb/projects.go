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

// Member represents a member of a project. It contains information
// for populating a member list in the UI, including the user's identity
// and their role in the project.
type Member struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Errors returned from project- and membership-related functions.
var (
	ErrBadProjectName = errors.New("invalid project name")
	ErrBadRole        = errors.New("invalid role")
	ErrAlreadyMember  = errors.New("user is already a member of the project")
)

// CreateProject creates a new project and adds the given user as its
// initial admin member.
func CreateProject(ctx context.Context, name string, userID string) (*Project, error) {
	// TODO(jeffomatic): the insert query and call to AddMember should be
	// wrapped in a database transaction. In order to do this, the pg package
	// should be updated so that tests do not fail when running operations that
	// require transactions.

	if name == "" {
		return nil, errors.Wrap(ErrBadProjectName)
	}

	var (
		q  = `INSERT INTO projects (name) VALUES ($1) RETURNING id`
		id string
	)
	err := pg.FromContext(ctx).QueryRow(ctx, q, name).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert query")
	}

	err = AddMember(ctx, id, userID, "admin")
	if err != nil {
		return nil, errors.Wrap(err, "add project creator as member")
	}

	return &Project{ID: id, Name: name}, nil
}

// ListProjects returns a list of projects that the given user is a
// member of.
func ListProjects(ctx context.Context, userID string) ([]*Project, error) {
	q := `
		SELECT p.id, p.name
		FROM projects p
		JOIN members m ON p.id = m.project_id
		WHERE m.user_id = $1
		ORDER BY p.name
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, userID)
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
	err := pg.FromContext(ctx).QueryRow(ctx, q, projID).Scan(&name)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "project id: %v", projID)
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
	err := pg.FromContext(ctx).QueryRow(ctx, q, name, projID).Scan(new(int))
	if err == sql.ErrNoRows {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "project ID: %v", projID)
	}
	if err != nil {
		return errors.Wrap(err, "update query")
	}
	return nil
}

// ListMembers returns a list of members of the given the given project.
// Member data includes each member's user information and their role within
// the project.
func ListMembers(ctx context.Context, projID string) ([]*Member, error) {
	q := `
		SELECT u.id, u.email, m.role
		FROM users u
		JOIN members m ON u.id = m.user_id
		WHERE m.project_id = $1
		ORDER BY u.email
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, projID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var members []*Member
	for rows.Next() {
		m := new(Member)
		err := rows.Scan(&m.ID, &m.Email, &m.Role)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return members, nil
}

// AddMember adds a new member to an project with a specific role. If the
// role is not valid, ErrBadRole will be returned. If the user is already a
// member of the project, ErrAlreadyMember is returned.
func AddMember(ctx context.Context, projID, userID, role string) error {
	if err := validateRole(role); err != nil {
		return err
	}

	q := `
		INSERT INTO members (project_id, user_id, role)
		SELECT $1, $2, $3
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, projID, userID, role)
	if pg.IsUniqueViolation(err) {
		return ErrAlreadyMember
	}
	if err != nil {
		return errors.Wrap(err, "insert query")
	}

	return nil
}

// UpdateMember changes the role of a user within an project. If the
// role is not valid, ErrBadRole will be returned. If the user is not a member
// of the project, an error with pg.ErrUserInputNotFound as its root will be
// returned.
func UpdateMember(ctx context.Context, projID, userID, role string) error {
	if err := validateRole(role); err != nil {
		return err
	}

	q := `
		UPDATE members SET role = $1
		WHERE project_id = $2 AND user_id = $3
		RETURNING 1
	`
	err := pg.FromContext(ctx).QueryRow(ctx, q, role, projID, userID).Scan(new(int))
	if err == sql.ErrNoRows {
		return errors.WithDetailf(
			pg.ErrUserInputNotFound,
			"project id: %v, user id: %v", projID, userID,
		)
	}
	if err != nil {
		return errors.Wrap(err, "update query")
	}
	return nil
}

// RemoveMember removes a member from the project.
func RemoveMember(ctx context.Context, projID string, userID string) error {
	q := `
		DELETE FROM members
		WHERE project_id = $1 AND user_id = $2
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, projID, userID)
	if err != nil {
		return errors.Wrap(err, "delete query")
	}
	return nil
}

// validateRole checks whether the provided role is one of the valid roles,
// either "admin" or "developer". If the role is invalid, an error with
// ErrBadRole as its root is returned.
func validateRole(role string) error {
	if role != "admin" && role != "developer" {
		return errors.WithDetailf(ErrBadRole, "role: %v", role)
	}
	return nil
}

// IsMember returns true if the user is a member of the project
func IsMember(ctx context.Context, userID string, project string) (bool, error) {
	const q = `
		SELECT COUNT(*)=1 FROM members WHERE user_id=$1 AND project_id=$2
	`
	var isMember bool
	row := pg.FromContext(ctx).QueryRow(ctx, q, userID, project)
	err := row.Scan(&isMember)
	return isMember, errors.Wrap(err)
}

// IsAdmin returns true if the user is an admin of the project
func IsAdmin(ctx context.Context, userID string, project string) (bool, error) {
	const q = `
		SELECT COUNT(*)=1 FROM members
		WHERE user_id=$1 AND project_id=$2 AND role='admin'
	`
	var isAdmin bool
	row := pg.FromContext(ctx).QueryRow(ctx, q, userID, project)
	err := row.Scan(&isAdmin)
	return isAdmin, errors.Wrap(err)
}

// ProjectByActiveManager returns the project ID associated with
// an active manager node
func ProjectByActiveManager(ctx context.Context, managerID string) (string, error) {
	const q = `
		SELECT project_id
		FROM manager_nodes WHERE id=$1 AND NOT archived
	`
	var project string
	err := pg.FromContext(ctx).QueryRow(ctx, q, managerID).Scan(&project)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	return project, errors.WithDetailf(err, "manager node %v", managerID)
}

// ProjectsByAccount returns all project IDs associated with a set of active accounts
func ProjectsByAccount(ctx context.Context, accountIDs ...string) ([]string, error) {
	// Remove duplicates so that we know how many accounts to expect.
	sort.Strings(accountIDs)
	accountIDs = strings.Uniq(accountIDs)

	const q = `
		SELECT COUNT(acc.id), array_agg(DISTINCT project_id) FROM accounts acc
		JOIN manager_nodes mn ON acc.manager_node_id=mn.id
		WHERE acc.id=ANY($1) AND NOT acc.archived
	`
	var (
		accountsFound int
		projects      []string
	)
	err := pg.FromContext(ctx).QueryRow(ctx, q, pg.Strings(accountIDs)).Scan(&accountsFound, (*pg.Strings)(&projects))
	if accountsFound != len(accountIDs) {
		err = pg.ErrUserInputNotFound
	}
	return projects, errors.Wrap(err)
}

// ProjectByActiveIssuer returns the project ID associated with an active issuer nodes
func ProjectByActiveIssuer(ctx context.Context, issuerID string) (string, error) {
	const q = `
		SELECT project_id
		FROM issuer_nodes WHERE id=$1 AND NOT archived
	`
	var project string
	err := pg.FromContext(ctx).QueryRow(ctx, q, issuerID).Scan(&project)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	return project, errors.WithDetailf(err, "issuer node %v", issuerID)
}

// ProjectByActiveAsset returns the project ID associated with an unarchived asset. If
// an asset has been archived, this function returns ErrArchived.
func ProjectByActiveAsset(ctx context.Context, assetID string) (string, error) {
	const q = `
		SELECT project_id, a.archived FROM assets a
		JOIN issuer_nodes i ON a.issuer_node_id=i.id
		WHERE a.id=$1
	`
	var (
		project  string
		archived bool
	)
	err := pg.FromContext(ctx).QueryRow(ctx, q, assetID).Scan(&project, &archived)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if archived {
		err = ErrArchived
	}
	return project, errors.WithDetailf(err, "asset %v", assetID)
}
