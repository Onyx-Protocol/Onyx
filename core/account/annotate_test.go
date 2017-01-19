package account

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"chain/core/query"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAnnotateTxs(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()
	acc1 := m.createTestAccount(ctx, t, "", nil)
	acc2 := m.createTestAccount(ctx, t, "", map[string]interface{}{"one": "foo", "two": "bar"})
	acp1 := m.createTestControlProgram(ctx, t, acc1.ID)
	acp2 := m.createTestControlProgram(ctx, t, acc2.ID)

	txs := []*query.AnnotatedTx{
		{
			Outputs: []*query.AnnotatedOutput{
				{ControlProgram: acp1},
				{ControlProgram: acp2},
			},
		},
	}
	empty := json.RawMessage(`{}`)
	wantTags := json.RawMessage(`{"one": "foo", "two": "bar"}`)
	want := []*query.AnnotatedTx{
		{
			Outputs: []*query.AnnotatedOutput{
				{Purpose: "receive", ControlProgram: acp1, AccountID: acc1.ID, AccountTags: &empty},
				{Purpose: "receive", ControlProgram: acp2, AccountID: acc2.ID, AccountTags: &wantTags},
			},
		},
	}

	err := m.AnnotateTxs(ctx, txs)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if !reflect.DeepEqual(txs, want) {
		t.Errorf("AnnotateTxs = %+v want %+v", txs, want)
	}
}
