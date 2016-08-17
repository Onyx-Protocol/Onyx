package mockhsm

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
)

type HSM struct {
	db pg.DB
}

type XPub struct {
	*hd25519.XPub
	Alias string
}

func New(db pg.DB) *HSM {
	return &HSM{db}
}

// CreateKey produces a new random xprv and stores it in the db.
func (h *HSM) CreateKey(ctx context.Context, alias string) (*XPub, error) {
	xprv, xpub, err := hd25519.NewXKeys(nil)
	if err != nil {
		return nil, err
	}
	hash := sha3.Sum256(xpub.Bytes())
	err = h.store(ctx, hex.EncodeToString(hash[:]), xprv, xpub, alias)
	if err != nil {
		return nil, err
	}
	return &XPub{XPub: xpub, Alias: alias}, nil
}

func (h *HSM) store(ctx context.Context, xpubHash string, xprv *hd25519.XPrv, xpub *hd25519.XPub, alias string) error {
	aliasSql := sql.NullString{
		String: alias,
		Valid:  alias != "",
	}
	_, err := h.db.Exec(ctx, "INSERT INTO mockhsm (xpub_hash, xpub, xprv, alias) VALUES ($1, $2, $3, $4)", xpubHash, xpub.Bytes(), xprv.Bytes(), aliasSql)
	return err
}

// ListKeys returns a list of all xpubs from the db.
func (h *HSM) ListKeys(ctx context.Context, cursor string, limit int) ([]*XPub, string, error) {
	var xpubs []*XPub
	const q = `
		SELECT xpub, alias FROM mockhsm
		WHERE ($1='' OR $1<xpub_hash)
		ORDER BY xpub_hash ASC LIMIT $2
	`
	err := pg.ForQueryRows(ctx, q, cursor, limit, func(b []byte, alias sql.NullString) {
		hdxpub, err := hd25519.XPubFromBytes(b)
		if err != nil {
			return
		}
		xpub := &XPub{XPub: hdxpub}
		if alias.Valid {
			xpub.Alias = alias.String
		}
		xpubs = append(xpubs, xpub)
	})
	if err != nil {
		return nil, "", err
	}

	var newCursor string
	if len(xpubs) > 0 {
		lastXPub := xpubs[len(xpubs)-1]
		hash := sha3.Sum256(lastXPub.Bytes())
		newCursor = hex.EncodeToString(hash[:])
	}
	return xpubs, newCursor, nil
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
