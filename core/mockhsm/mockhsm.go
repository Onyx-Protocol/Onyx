package mockhsm

import (
	"context"
	"database/sql"
	"strconv"
	"sync"

	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/errors"
)

var (
	ErrDuplicateKeyAlias = errors.New("duplicate key alias")
	ErrInvalidAfter      = errors.New("invalid after")
)

type HSM struct {
	db pg.DB

	kdCacheMu sync.Mutex
	kdCache   map[chainkd.XPub]chainkd.XPrv
}

type XPub struct {
	Alias *string      `json:"alias"`
	XPub  chainkd.XPub `json:"xpub"`
}

func New(db pg.DB) *HSM {
	return &HSM{db: db, kdCache: make(map[chainkd.XPub]chainkd.XPrv)}
}

// CreateChainKDKey produces a new random xprv and stores it in the db.
func (h *HSM) CreateChainKDKey(ctx context.Context, alias string) (*XPub, error) {
	xpub, _, err := h.createChainKDKey(ctx, alias, false)
	return xpub, err
}

// GetOrCreateChainKDKey looks for the ChainKD key with the given alias, generating a
// new one if it's not found.
func (h *HSM) GetOrCreateChainKDKey(ctx context.Context, alias string) (xpub *XPub, created bool, err error) {
	return h.createChainKDKey(ctx, alias, true)
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
	const q = `INSERT INTO mockhsm (pub, prv, alias) VALUES ($1, $2, $3)`
	_, err = h.db.Exec(ctx, q, xpub.Bytes(), xprv.Bytes(), sqlAlias)
	if err != nil {
		if pg.IsUniqueViolation(err) {
			if !get {
				return nil, false, errors.WithDetailf(ErrDuplicateKeyAlias, "value: %q", alias)
			}

			var xpubBytes []byte
			err = pg.QueryRow(ctx, `SELECT pub FROM mockhsm WHERE alias = $1`, alias).Scan(&xpubBytes)
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
		SELECT pub, alias, sort_id FROM mockhsm
		WHERE ($1=0 OR $1 < sort_id)
		ORDER BY sort_id DESC LIMIT $2
	`
	err = pg.ForQueryRows(ctx, q, zafter, limit, func(b []byte, alias sql.NullString, sortID int64) {
		var hdxpub chainkd.XPub
		copy(hdxpub[:], b)
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

func (h *HSM) loadChainKDKey(ctx context.Context, xpub chainkd.XPub) (xprv chainkd.XPrv, err error) {
	h.kdCacheMu.Lock()
	defer h.kdCacheMu.Unlock()

	if xprv, ok := h.kdCache[xpub]; ok {
		return xprv, nil
	}

	var b []byte
	err = h.db.QueryRow(ctx, "SELECT prv FROM mockhsm WHERE pub = $1", xpub.Bytes()).Scan(&b)
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

// SignWithChainKDKey looks up the xprv given the xpub, optionally derives a new
// xprv with the given path (but does not store the new xprv), and
// signs the given msg.
func (h *HSM) SignWithChainKDKey(ctx context.Context, xpub chainkd.XPub, path [][]byte, msg []byte) ([]byte, error) {
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
	h.kdCacheMu.Lock()
	delete(h.kdCache, xpub)
	h.kdCacheMu.Unlock()
	_, err := h.db.Exec(ctx, "DELETE FROM mockhsm WHERE pub = $1", xpub.Bytes())
	return err
}
