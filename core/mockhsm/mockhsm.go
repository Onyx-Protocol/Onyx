package mockhsm

import (
	"context"
	"database/sql"
	"encoding/hex"
	"strconv"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
)

var (
	ErrDuplicateKeyAlias = errors.New("duplicate key alias")
	ErrInvalidAfter      = errors.New("invalid after")
)

type HSM struct {
	db pg.DB
}

type XPub struct {
	Alias *string       `json:"alias"`
	XPub  *hd25519.XPub `json:"xpub"`
}

func New(db pg.DB) *HSM {
	return &HSM{db}
}

// CreateKey produces a new random xprv and stores it in the db.
func (h *HSM) CreateKey(ctx context.Context, alias *string) (*XPub, error) {
	xpub, _, err := h.create(ctx, alias, false)
	return xpub, err
}

// GetOrCreateKey looks for the key with the given alias, generating a
// new one if it's not found.
func (h *HSM) GetOrCreateKey(ctx context.Context, alias string) (xpub *XPub, created bool, err error) {
	return h.create(ctx, &alias, true)
}

func (h *HSM) create(ctx context.Context, alias *string, get bool) (*XPub, bool, error) {
	xprv, xpub, err := hd25519.NewXKeys(nil)
	if err != nil {
		return nil, false, err
	}
	hash := sha3.Sum256(xpub.Bytes())
	const q = `INSERT INTO mockhsm (xpub_hash, xpub, xprv, alias) VALUES ($1, $2, $3, $4)`
	_, err = h.db.Exec(ctx, q, hex.EncodeToString(hash[:]), xpub.Bytes(), xprv.Bytes(), alias)
	if err != nil {
		if pg.IsUniqueViolation(err) {
			if !get {
				return nil, false, errors.WithDetailf(ErrDuplicateKeyAlias, "value: %q", alias)
			}

			var xpubBytes []byte
			err = pg.QueryRow(ctx, `SELECT xpub FROM mockhsm WHERE alias = $1`, alias).Scan(&xpubBytes)
			if err != nil {
				return nil, false, errors.Wrapf(err, "reading existing xpub with alias %s", alias)
			}
			existingXPub, err := hd25519.XPubFromBytes(xpubBytes)
			if err != nil {
				return nil, false, errors.Wrapf(err, "parsing bytes of existing xpub with alias %s", alias)
			}
			return &XPub{XPub: existingXPub, Alias: alias}, false, nil
		}
		return nil, false, errors.Wrap(err, "storing new xpub")
	}
	return &XPub{XPub: xpub, Alias: alias}, true, nil
}

// ListKeys returns a list of all xpubs from the db.
func (h *HSM) ListKeys(ctx context.Context, after string, limit int) ([]*XPub, string, error) {
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

	var xpubs []*XPub
	const q = `
		SELECT xpub, alias, sort_id FROM mockhsm
		WHERE ($1=0 OR $1 < sort_id)
		ORDER BY sort_id DESC LIMIT $2
	`
	err = pg.ForQueryRows(ctx, q, zafter, limit, func(b []byte, alias sql.NullString, sortID int64) {
		hdxpub, err := hd25519.XPubFromBytes(b)
		if err != nil {
			return
		}
		xpub := &XPub{XPub: hdxpub}
		if alias.Valid {
			xpub.Alias = &alias.String
		}
		xpubs = append(xpubs, xpub)
		zafter = sortID
	})
	if err != nil {
		return nil, "", err
	}

	return xpubs, strconv.FormatInt(zafter, 10), nil
}

var ErrNoKey = errors.New("key not found")

func (h *HSM) load(ctx context.Context, xpub *hd25519.XPub) (*hd25519.XPrv, error) {
	var b []byte
	err := h.db.QueryRow(ctx, "SELECT xprv FROM mockhsm WHERE xpub = $1", xpub.Bytes()).Scan(&b)
	if err == sql.ErrNoRows {
		return nil, ErrNoKey
	}
	if err != nil {
		return nil, err
	}
	return hd25519.XPrvFromBytes(b)
}

// Sign looks up the xprv given the xpub, optionally derives a new
// xprv with the given path (but does not store the new xprv), and
// signs the given msg.
func (h *HSM) Sign(ctx context.Context, xpub *hd25519.XPub, path []uint32, msg []byte) ([]byte, error) {
	xprv, err := h.load(ctx, xpub)
	if err != nil {
		return nil, err
	}
	if len(path) > 0 {
		xprv = xprv.Derive(path)
	}
	return xprv.Sign(msg), nil
}

func (h *HSM) DelKey(ctx context.Context, xpub *hd25519.XPub) error {
	_, err := h.db.Exec(ctx, "DELETE FROM mockhsm WHERE xpub = $1", xpub.Bytes())
	return err
}
