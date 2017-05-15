// Package mockhsm provides a mock HSM for development environments.
// It is unsafe for use in production.
package mockhsm

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"

	"github.com/lib/pq"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol/bc/legacy"
)

// listKeyMaxAliases limits the alias filter to a sane maximum size.
const listKeyMaxAliases = 200

var (
	ErrDuplicateKeyAlias    = errors.New("duplicate key alias")
	ErrInvalidAfter         = errors.New("invalid after")
	ErrNoKey                = errors.New("key not found")
	ErrInvalidKeySize       = errors.New("key invalid size")
	ErrTooManyAliasesToList = errors.New("requested aliases exceeds limit")
)

type HSM struct {
	db pg.DB

	cacheMu sync.Mutex
	kdCache map[chainkd.XPub]chainkd.XPrv
	edCache map[string]ed25519.PrivateKey // ed25519.PublicKeys must be turned into strings before being used as map keys
}

type XPub struct {
	Alias *string      `json:"alias"`
	XPub  chainkd.XPub `json:"xpub"`
}

type Pub struct {
	Alias *string           `json:"alias"`
	Pub   ed25519.PublicKey `json:"pub"`
}

func New(db pg.DB) *HSM {
	return &HSM{
		db:      db,
		kdCache: make(map[chainkd.XPub]chainkd.XPrv),
		edCache: make(map[string]ed25519.PrivateKey),
	}
}

// XCreate produces a new random xprv and stores it in the db.
func (h *HSM) XCreate(ctx context.Context, alias string) (*XPub, error) {
	xpub, _, err := h.createChainKDKey(ctx, alias, false)
	return xpub, err
}

func (h *HSM) createChainKDKey(ctx context.Context, alias string, get bool) (*XPub, bool, error) {
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		return nil, false, err
	}
	sqlAlias := sql.NullString{String: alias, Valid: alias != ""}
	var ptrAlias *string
	if alias != "" {
		ptrAlias = &alias
	}
	const q = `INSERT INTO mockhsm (pub, prv, alias, key_type) VALUES ($1, $2, $3, 'chain_kd')`
	_, err = h.db.ExecContext(ctx, q, xpub.Bytes(), xprv.Bytes(), sqlAlias)
	if err != nil {
		if pg.IsUniqueViolation(err) {
			if !get {
				return nil, false, errors.WithDetailf(ErrDuplicateKeyAlias, "value: %q", alias)
			}

			var xpubBytes []byte
			err = h.db.QueryRowContext(ctx, `SELECT pub FROM mockhsm WHERE alias = $1`, alias).Scan(&xpubBytes)
			if err != nil {
				return nil, false, errors.Wrapf(err, "reading existing xpub with alias %s", alias)
			}
			var existingXPub chainkd.XPub
			copy(existingXPub[:], xpubBytes)
			return &XPub{XPub: existingXPub, Alias: ptrAlias}, false, nil
		}
		return nil, false, errors.Wrap(err, "storing new xpub")
	}
	return &XPub{XPub: xpub, Alias: ptrAlias}, true, nil
}

// Create produces a new random prv and stores it in the db.
func (h *HSM) Create(ctx context.Context, alias string) (*Pub, error) {
	pub, _, err := h.createEd25519Key(ctx, alias, false)
	return pub, err
}

// GetOrCreate looks for the Ed25519 key with the given alias, generating a
// new one if it's not found.
func (h *HSM) GetOrCreate(ctx context.Context, alias string) (*Pub, bool, error) {
	return h.createEd25519Key(ctx, alias, true)
}

func (h *HSM) createEd25519Key(ctx context.Context, alias string, get bool) (*Pub, bool, error) {
	pub, prv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, false, err
	}

	sqlAlias := sql.NullString{String: alias, Valid: alias != ""}
	var ptrAlias *string
	if alias != "" {
		ptrAlias = &alias
	}
	const q = `INSERT INTO mockhsm (pub, prv, alias, key_type) VALUES ($1, $2, $3, 'ed25519')`
	_, err = h.db.ExecContext(ctx, q, []byte(pub), []byte(prv), sqlAlias)
	if err != nil {
		if pg.IsUniqueViolation(err) {
			if !get {
				return nil, false, errors.WithDetailf(ErrDuplicateKeyAlias, "value: %q", alias)
			}

			var pubBytes []byte
			err = h.db.QueryRowContext(ctx, `SELECT pub FROM mockhsm WHERE alias = $1`, alias).Scan(&pubBytes)
			if err != nil {
				return nil, false, errors.Wrapf(err, "reading existing pub with alias %s", alias)
			}
			return &Pub{Pub: ed25519.PublicKey(pubBytes), Alias: ptrAlias}, false, nil
		}
		return nil, false, errors.Wrap(err, "storing new pub")
	}
	return &Pub{Pub: pub, Alias: ptrAlias}, true, nil
}

// ListKeys returns a list of all xpubs from the db.
func (h *HSM) ListKeys(ctx context.Context, aliases []string, after string, limit int) ([]*XPub, string, error) {
	if len(aliases) > listKeyMaxAliases {
		return nil, "", errors.WithDetailf(ErrTooManyAliasesToList, "max: %d", listKeyMaxAliases)
	}

	var (
		zafter int64
		err    error
	)

	if after != "" {
		zafter, err = strconv.ParseInt(after, 10, 64)
		if err != nil {
			return nil, "", errors.WithDetailf(ErrInvalidAfter, "value: %q", after)
		}
	}

	var (
		xpubs  []*XPub
		params []interface{}
	)
	q := `
		SELECT pub, alias, sort_id FROM mockhsm
		WHERE key_type = 'chain_kd'
	`

	if len(aliases) > 0 {
		params = append(params, pq.StringArray(aliases))
		q += fmt.Sprintf(" AND alias = ANY($%d)", len(params))
	}

	if zafter != 0 {
		params = append(params, zafter)
		q += fmt.Sprintf(" AND sort_id < $%d", len(params))
	}

	q += fmt.Sprintf(" ORDER BY sort_id DESC LIMIT %d", limit)

	consumeRow := func(b []byte, alias sql.NullString, sortID int64) {
		var hdxpub chainkd.XPub
		copy(hdxpub[:], b)
		xpub := &XPub{XPub: hdxpub}
		if alias.Valid {
			xpub.Alias = &alias.String
		}
		xpubs = append(xpubs, xpub)
		zafter = sortID
	}
	params = append(params, consumeRow)

	err = pg.ForQueryRows(ctx, h.db, q, params...)
	if err != nil {
		return nil, "", err
	}

	return xpubs, strconv.FormatInt(zafter, 10), nil
}

func (h *HSM) loadChainKDKey(ctx context.Context, xpub chainkd.XPub) (xprv chainkd.XPrv, err error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	if xprv, ok := h.kdCache[xpub]; ok {
		return xprv, nil
	}

	var b []byte
	err = h.db.QueryRowContext(ctx, "SELECT prv FROM mockhsm WHERE pub = $1 AND key_type='chain_kd'", xpub.Bytes()).Scan(&b)
	if err == sql.ErrNoRows {
		return xprv, ErrNoKey
	}
	if err != nil {
		return xprv, err
	}
	copy(xprv[:], b)
	h.kdCache[xpub] = xprv
	return xprv, nil
}

// XSign looks up the xprv given the xpub, optionally derives a new
// xprv with the given path (but does not store the new xprv), and
// signs the given msg.
func (h *HSM) XSign(ctx context.Context, xpub chainkd.XPub, path [][]byte, msg []byte) ([]byte, error) {
	xprv, err := h.loadChainKDKey(ctx, xpub)
	if err != nil {
		return nil, err
	}
	if len(path) > 0 {
		xprv = xprv.Derive(path)
	}
	return xprv.Sign(msg), nil
}

func (h *HSM) DeleteChainKDKey(ctx context.Context, xpub chainkd.XPub) error {
	h.cacheMu.Lock()
	delete(h.kdCache, xpub)
	h.cacheMu.Unlock()
	_, err := h.db.ExecContext(ctx, "DELETE FROM mockhsm WHERE pub = $1 AND key_type='chain_kd'", xpub.Bytes())
	return err
}

func (h *HSM) loadEd25519Key(ctx context.Context, pub ed25519.PublicKey) (prv ed25519.PrivateKey, err error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	pubStr := string(pub)

	if prv, ok := h.edCache[pubStr]; ok {
		return prv, nil
	}

	err = h.db.QueryRowContext(ctx, "SELECT prv FROM mockhsm WHERE pub = $1 AND key_type='ed25519'", []byte(pub)).Scan(&prv)
	if err == sql.ErrNoRows {
		return prv, ErrNoKey
	}
	if err != nil {
		return prv, err
	}
	h.edCache[pubStr] = prv
	return prv, nil
}

// Sign looks up the prv given the pub and signs the given msg.
func (h *HSM) Sign(ctx context.Context, pub ed25519.PublicKey, bh *legacy.BlockHeader) ([]byte, error) {
	prv, err := h.loadEd25519Key(ctx, pub)
	if err != nil {
		return nil, err
	}

	// ed25519.Sign will panic if prv is the wrong size. Protect against that.
	if len(prv) != ed25519.PrivateKeySize {
		return nil, ErrInvalidKeySize
	}
	msg := bh.Hash()
	return ed25519.Sign(prv, msg.Bytes()), nil
}
