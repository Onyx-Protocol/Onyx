package account

import (
	"context"
	"encoding/json"
	"testing"

	"chain/core/query"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAnnotateTxs(t *testing.T) {
	var (
		db   = pgtest.NewTx(t)
		m    = NewManager(db, prottest.NewChain(t), nil)
		ctx  = context.Background()
		acc1 = m.createTestAccount(ctx, t, "", nil)
		acc2 = m.createTestAccount(ctx, t, "", map[string]interface{}{"one": "foo", "two": "bar"})
		u1   = m.createTestUTXO(ctx, t, acc1.ID)
		u2   = m.createTestUTXO(ctx, t, acc2.ID)
		u3   = m.createTestUTXO(ctx, t, acc2.ID)
	)

	txs := []*query.AnnotatedTx{
		{
			Outputs: []*query.AnnotatedOutput{
				{OutputID: u1},
				{OutputID: u2},
				{OutputID: u3},
			},
		},
	}
	empty := json.RawMessage(`{}`)
	wantTags := json.RawMessage(`{"one": "foo", "two": "bar"}`)
	want := []*query.AnnotatedTx{
		{
			Outputs: []*query.AnnotatedOutput{
				{Purpose: "receive", OutputID: u1, AccountID: acc1.ID, AccountTags: &empty},
				{Purpose: "receive", OutputID: u2, AccountID: acc2.ID, AccountTags: &wantTags},
				{Purpose: "receive", OutputID: u3, AccountID: acc2.ID, AccountTags: &wantTags},
			},
		},
	}

	err := m.AnnotateTxs(ctx, txs)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if !testutil.DeepEqual(txs, want) {
		t.Errorf("AnnotateTxs = %+v want %+v", txs, want)
	}
}
