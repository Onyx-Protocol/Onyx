package signers

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
)

type keySpace uint32

const (
	AssetKeySpace   keySpace = 0
	AccountKeySpace keySpace = 1
)

var typeIDMap = map[string]string{
	"account": "acc",
	"asset":   "asset",
}

var (
	// ErrBadQuorum is returned by Create when the quorum
	// provided is less than 1 or greater than the number
	// of xpubs provided.
	ErrBadQuorum = errors.New("quorum must be greater than 1 and less than or equal to the length of xpubs")

	// ErrBadXPub is returned by Create when the xpub
	// provided isn't valid.
	ErrBadXPub = errors.New("invalid xpub format")

	// ErrNoXPubs is returned by create when the xpubs
	// slice provided is empty.
	ErrNoXPubs = errors.New("at least one xpub is required")

	// ErrBadType is returned when a find operation
	// retrieves a signer that is not the expected type.
	ErrBadType = errors.New("retrieved type does not match expected type")

	// ErrArchived is return when a find operation
	// retrieves a signer that has been archived.
	ErrArchived = errors.New("archived")
)

// Signer is the abstract concept of a signer,
// which is composed of a set of keys as well as
// the amount of signatures needed for quorum.
type Signer struct {
	ID       string
	Type     string
	XPubs    []*hd25519.XPub
	Quorum   int
	KeyIndex []uint32
}

// Path returns the complete path for derived keys
func Path(s *Signer, ks keySpace, itemIndex []uint32) []uint32 {
	return append(append([]uint32{uint32(ks)}, s.KeyIndex...), itemIndex...)
}

// Create creates and stores a Signer in the database
func Create(ctx context.Context, typ string, xpubs []string, quorum int, clientToken *string) (*Signer, error) {
	if len(xpubs) == 0 {
		return nil, errors.Wrap(ErrNoXPubs)
	}

	keys, err := ConvertKeys(xpubs)
	if err != nil {
		return nil, err
	}

	if quorum == 0 || quorum > len(xpubs) {
		return nil, errors.Wrap(ErrBadQuorum)
	}

	const q = `
		INSERT INTO signers (id, type, xpubs, quorum, client_token)
		VALUES (next_chain_id($1::text), $2, $3, $4, $5)
		ON CONFLICT (client_token) DO NOTHING
		RETURNING id, key_index(key_index)
  `
	var (
		id       string
		keyIndex []uint32
	)
	err = pg.QueryRow(ctx, q, typeIDMap[typ], typ, pq.StringArray(xpubs), quorum, clientToken).
		Scan(&id, (*pg.Uint32s)(&keyIndex))
	if err == sql.ErrNoRows && clientToken != nil {
		return findByClientToken(ctx, clientToken)
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err)
	}

	return &Signer{
		ID:       id,
		Type:     typ,
		XPubs:    keys,
		Quorum:   quorum,
		KeyIndex: keyIndex,
	}, nil
}

func findByClientToken(ctx context.Context, clientToken *string) (*Signer, error) {
	const q = `
		SELECT id, type, xpubs, quorum, key_index(key_index)
		FROM signers WHERE client_token=$1
	`

	var (
		s        Signer
		xpubStrs []string
	)
	err := pg.QueryRow(ctx, q, clientToken).
		Scan(&s.ID, &s.Type, (*pq.StringArray)(&xpubStrs), &s.Quorum, (*pg.Uint32s)(&s.KeyIndex))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	keys, err := ConvertKeys(xpubStrs)
	if err != nil {
		return nil, errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
	}

	s.XPubs = keys

	return &s, nil
}

// Find retrieves a Signer from the database
// using the type and id.
func Find(ctx context.Context, typ, id string) (*Signer, error) {
	const q = `
		SELECT id, type, xpubs, quorum, key_index(key_index), archived
		FROM signers WHERE id=$1
	`

	var (
		s        Signer
		archived bool
		xpubStrs []string
	)
	err := pg.QueryRow(ctx, q, id).Scan(
		&s.ID,
		&s.Type,
		(*pq.StringArray)(&xpubStrs),
		&s.Quorum,
		(*pg.Uint32s)(&s.KeyIndex),
		&archived,
	)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if archived {
		return nil, errors.Wrap(ErrArchived)
	}

	if s.Type != typ {
		return nil, errors.Wrap(ErrBadType)
	}

	keys, err := ConvertKeys(xpubStrs)
	if err != nil {
		return nil, errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
	}

	s.XPubs = keys

	return &s, nil
}

// Archive marks a Signer as archived in the database
func Archive(ctx context.Context, typ, id string) error {
	const q = `
		UPDATE signers SET archived='t' WHERE type=$1 and id=$2
		RETURNING id
	`
	err := pg.QueryRow(ctx, q, typ, id).Scan(&id)
	if err == sql.ErrNoRows {
		return errors.Wrap(pg.ErrUserInputNotFound)
	}
	return errors.Wrap(err)
}

// List returns a paginated set of Signers, limited to
// the provided type.
func List(ctx context.Context, typ, prev string, limit int) ([]*Signer, string, error) {
	const q = `
		SELECT id, type, xpubs, quorum, key_index(key_index)
		FROM signers WHERE type=$1 AND ($2='' OR $2<id)
		ORDER BY id ASC LIMIT $3
	`

	var signers []*Signer
	err := pg.ForQueryRows(ctx, q, typ, prev, limit,
		func(id, typ string, xpubs pq.StringArray, quorum int, keyIndex pg.Uint32s) error {
			keys, err := ConvertKeys(xpubs)
			if err != nil {
				return errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
			}

			signers = append(signers, &Signer{
				ID:       id,
				Type:     typ,
				XPubs:    keys,
				Quorum:   quorum,
				KeyIndex: keyIndex,
			})
			return nil
		},
	)

	if err != nil {
		return nil, "", errors.Wrap(err)
	}

	var last string
	if len(signers) > 0 {
		last = signers[len(signers)-1].ID
	}

	return signers, last, nil
}

func ConvertKeys(xpubs []string) ([]*hd25519.XPub, error) {
	var xkeys []*hd25519.XPub
	for i, xpub := range xpubs {
		xkey, err := hd25519.XPubFromString(xpub)
		if err != nil {
			return nil, errors.WithDetailf(ErrBadXPub, "key %d: xpub is not valid", i)
		}
		xkeys = append(xkeys, xkey)
	}
	return xkeys, nil
}
