package appdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

// Application represents an application. It can be used safely for API
// responses.
type Application struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Member represents a member of an application. It contains information
// for populating a member list in the UI, including the user's identity
// and their role in the application.
type Member struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Errors returned from application- and membership-related functions.
var (
	ErrBadRole       = errors.New("invalid role")
	ErrAlreadyMember = errors.New("user is already a member of the application")
)

// CreateApplication creates a new application and adds the given user as its
// initial admin member.
func CreateApplication(ctx context.Context, name string, userID string) (*Application, error) {
	// TODO(jeffomatic): the insert query and call to AddMember should be
	// wrapped in a database transaction. In order to do this, the pg package
	// should be updated so that tests do not fail when running operations that
	// require transactions.

	var (
		q  = `INSERT INTO applications (name) VALUES ($1) RETURNING id`
		id string
	)
	err := pg.FromContext(ctx).QueryRow(q, name).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert query")
	}

	err = AddMember(ctx, id, userID, "admin")
	if err != nil {
		return nil, errors.Wrap(err, "add app creator as member")
	}

	return &Application{ID: id, Name: name}, nil
}

// ListApplications returns a list of applications that the given user is a
// member of.
func ListApplications(ctx context.Context, userID string) ([]*Application, error) {
	q := `
		SELECT a.id, a.name
		FROM applications a
		JOIN members m ON a.id = m.application_id
		WHERE m.user_id = $1
		ORDER BY a.name
	`
	rows, err := pg.FromContext(ctx).Query(q, userID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var apps []*Application
	for rows.Next() {
		a := new(Application)
		err := rows.Scan(&a.ID, &a.Name)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		apps = append(apps, a)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return apps, nil
}

// GetApplication returns information about a single application.
func GetApplication(ctx context.Context, appID string) (*Application, error) {
	var (
		q    = `SELECT name FROM applications WHERE id = $1`
		name string
	)
	err := pg.FromContext(ctx).QueryRow(q, appID).Scan(&name)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "application id: %v", appID)
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}

	return &Application{ID: appID, Name: name}, nil
}

// ListMembers returns a list of members of the given the given application.
// Member data includes each member's user information and their role within
// the application.
func ListMembers(ctx context.Context, appID string) ([]*Member, error) {
	q := `
		SELECT u.id, u.email, m.role
		FROM users u
		JOIN members m ON u.id = m.user_id
		WHERE m.application_id = $1
		ORDER BY u.email
	`
	rows, err := pg.FromContext(ctx).Query(q, appID)
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

// AddMember adds a new member to an application with a specific role. If the
// role is not valid, ErrBadRole will be returned. If the user is already a
// member of the application, ErrAlreadyMember is returned.
func AddMember(ctx context.Context, appID, userID, role string) error {
	if err := validateRole(role); err != nil {
		return err
	}

	q := `
		INSERT INTO members (application_id, user_id, role)
		SELECT $1, $2, $3
	`
	_, err := pg.FromContext(ctx).Exec(q, appID, userID, role)
	if pg.IsUniqueViolation(err) {
		return ErrAlreadyMember
	}
	if err != nil {
		return errors.Wrap(err, "insert query")
	}

	return nil
}

// UpdateMember changes the role of a user within an application. If the
// role is not valid, ErrBadRole will be returned. If the user is not a member
// of the application, an error with pg.ErrUserInputNotFound as its root will be
// returned.
func UpdateMember(ctx context.Context, appID, userID, role string) error {
	if err := validateRole(role); err != nil {
		return err
	}

	q := `
		UPDATE members SET role = $1
		WHERE application_id = $2 AND user_id = $3
		RETURNING 1
	`
	err := pg.FromContext(ctx).QueryRow(q, role, appID, userID).Scan(new(int))
	if err == sql.ErrNoRows {
		return errors.WithDetailf(
			pg.ErrUserInputNotFound,
			"application id: %v, user id: %v", appID, userID,
		)
	}
	if err != nil {
		return errors.Wrap(err, "update query")
	}
	return nil
}

// RemoveMember removes a member from the application.
func RemoveMember(ctx context.Context, appID string, userID string) error {
	q := `
		DELETE FROM members
		WHERE application_id = $1 AND user_id = $2
	`
	_, err := pg.FromContext(ctx).Exec(q, appID, userID)
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
