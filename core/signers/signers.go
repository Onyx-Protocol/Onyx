// Package signers associates signers and their corresponding keys.
package signers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"sort"

	"github.com/lib/pq"

	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/errors"
)

type keySpace byte

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

	// ErrDupeXPub is returned by create when the same xpub
	// appears twice in a single call.
	ErrDupeXPub = errors.New("xpubs cannot contain the same key more than once")
)

// Signer is the abstract concept of a signer,
// which is composed of a set of keys as well as
// the amount of signatures needed for quorum.
type Signer struct {
	ID       string
	Type     string
	XPubs    []chainkd.XPub
	Quorum   int
	KeyIndex uint64
}

// Path returns the complete path for derived keys
func Path(s *Signer, ks keySpace, itemIndexes ...uint64) [][]byte {
	var path [][]byte
	signerPath := [9]byte{byte(ks)}
	binary.LittleEndian.PutUint64(signerPath[1:], s.KeyIndex)
	path = append(path, signerPath[:])
	for _, idx := range itemIndexes {
		var idxBytes [8]byte
		binary.LittleEndian.PutUint64(idxBytes[:], idx)
		path = append(path, idxBytes[:])
	}
	return path
}

// Create creates and stores a Signer in the database
func Create(ctx context.Context, db pg.DB, typ string, xpubs []chainkd.XPub, quorum int, clientToken string) (*Signer, error) {
	if len(xpubs) == 0 {
		return nil, errors.Wrap(ErrNoXPubs)
	}

	sort.Sort(sortKeys(xpubs)) // this transforms the input slice
	for i := 1; i < len(xpubs); i++ {
		if bytes.Equal(xpubs[i][:], xpubs[i-1][:]) {
			return nil, errors.WithDetailf(ErrDupeXPub, "duplicated key=%x", xpubs[i])
		}
	}

	if quorum == 0 || quorum > len(xpubs) {
		return nil, errors.Wrap(ErrBadQuorum)
	}

	var xpubBytes [][]byte
	for _, key := range xpubs {
		key := key
		xpubBytes = append(xpubBytes, key[:])
	}

	nullToken := sql.NullString{
		String: clientToken,
		Valid:  clientToken != "",
	}

	const q = `
		INSERT INTO signers (id, type, xpubs, quorum, client_token)
		VALUES (next_chain_id($1::text), $2, $3, $4, $5)
		ON CONFLICT (client_token) DO NOTHING
		RETURNING id, key_index
  `
	var (
		id       string
		keyIndex uint64
	)
	err := db.QueryRowContext(ctx, q, typeIDMap[typ], typ, pq.ByteaArray(xpubBytes), quorum, nullToken).
		Scan(&id, &keyIndex)
	if err == sql.ErrNoRows && clientToken != "" {
		return findByClientToken(ctx, db, clientToken)
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err)
	}

	return &Signer{
		ID:       id,
		Type:     typ,
		XPubs:    xpubs,
		Quorum:   quorum,
		KeyIndex: keyIndex,
	}, nil
}

func New(id, typ string, xpubs [][]byte, quorum int, keyIndex uint64) (*Signer, error) {
	keys, err := ConvertKeys(xpubs)
	if err != nil {
		return nil, errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
	}
	return &Signer{
		ID:       id,
		Type:     typ,
		XPubs:    keys,
		Quorum:   quorum,
		KeyIndex: keyIndex,
	}, nil
}

func findByClientToken(ctx context.Context, db pg.DB, clientToken string) (*Signer, error) {
	const q = `
		SELECT id, type, xpubs, quorum, key_index
		FROM signers WHERE client_token=$1
	`

	var (
		s         Signer
		xpubBytes [][]byte
	)
	err := db.QueryRowContext(ctx, q, clientToken).
		Scan(&s.ID, &s.Type, (*pq.ByteaArray)(&xpubBytes), &s.Quorum, &s.KeyIndex)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	keys, err := ConvertKeys(xpubBytes)
	if err != nil {
		return nil, errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
	}

	s.XPubs = keys

	return &s, nil
}

// Find retrieves a Signer from the database
// using the type and id.
func Find(ctx context.Context, db pg.DB, typ, id string) (*Signer, error) {
	const q = `
		SELECT id, type, xpubs, quorum, key_index
		FROM signers WHERE id=$1
	`

	var (
		s         Signer
		xpubBytes [][]byte
	)
	err := db.QueryRowContext(ctx, q, id).Scan(
		&s.ID,
		&s.Type,
		(*pq.ByteaArray)(&xpubBytes),
		&s.Quorum,
		&s.KeyIndex,
	)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if s.Type != typ {
		return nil, errors.Wrap(ErrBadType)
	}

	keys, err := ConvertKeys(xpubBytes)
	if err != nil {
		return nil, errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
	}

	s.XPubs = keys

	return &s, nil
}

// List returns a paginated set of Signers, limited to
// the provided type.
func List(ctx context.Context, db pg.DB, typ, prev string, limit int) ([]*Signer, string, error) {
	const q = `
		SELECT id, type, xpubs, quorum, key_index
		FROM signers WHERE type=$1 AND ($2='' OR $2<id)
		ORDER BY id ASC LIMIT $3
	`

	var signers []*Signer
	err := pg.ForQueryRows(ctx, db, q, typ, prev, limit,
		func(id, typ string, xpubs pq.ByteaArray, quorum int, keyIndex uint64) error {
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

func ConvertKeys(xpubs [][]byte) ([]chainkd.XPub, error) {
	var xkeys []chainkd.XPub
	for i, xpub := range xpubs {
		var xkey chainkd.XPub
		if len(xpub) != len(xkey) {
			return nil, errors.WithDetailf(ErrBadXPub, "key %d: xpub is not valid", i)
		}
		copy(xkey[:], xpub)
		xkeys = append(xkeys, xkey)
	}
	return xkeys, nil
}

type sortKeys []chainkd.XPub

func (s sortKeys) Len() int           { return len(s) }
func (s sortKeys) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s sortKeys) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
