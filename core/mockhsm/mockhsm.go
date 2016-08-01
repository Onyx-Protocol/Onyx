package mockhsm

import (
	"golang.org/x/net/context"

	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
)

type HSM struct {
	db pg.DB
}

func New(db pg.DB) *HSM {
	return &HSM{db}
}

// GenXPrv produces a new random xprv and stores it in the db.
func (h *HSM) GenKey(ctx context.Context) (*hd25519.XPub, error) {
	xprv, xpub, err := hd25519.NewXKeys(nil)
	if err != nil {
		return nil, err
	}
	err = h.store(ctx, xprv, xpub)
	if err != nil {
		return nil, err
	}
	return xpub, nil
}

func (h *HSM) store(ctx context.Context, xprv *hd25519.XPrv, xpub *hd25519.XPub) error {
	_, err := h.db.Exec(ctx, "INSERT INTO mockhsm (xpub, xprv) VALUES ($1, $2)", xpub.Bytes(), xprv.Bytes())
	return err
}

func (h *HSM) load(ctx context.Context, xpub *hd25519.XPub) (*hd25519.XPrv, error) {
	var b []byte
	err := h.db.QueryRow(ctx, "SELECT xprv FROM mockhsm WHERE xpub = $1", xpub.Bytes()).Scan(&b)
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
