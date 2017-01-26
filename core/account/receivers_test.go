package account

import (
	"context"
	"testing"
	"time"

	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestCreateReceiver(t *testing.T) {
	// use pgtest.NewDB for deterministic postgres sequences
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "alias", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	exp := time.Now().Add(24 * 365 * time.Hour)
	_, err = m.CreateReceiver(ctx, account.ID, "", exp)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = m.CreateReceiver(ctx, "", "alias", exp)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
