package account

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"chain-stealth/core/confidentiality"
	"chain-stealth/database/pg/pgtest"
	"chain-stealth/protocol/prottest"
	"chain-stealth/testutil"
)

func TestAnnotateTxs(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	m := NewManager(db, prottest.NewChain(t), nil, &confidentiality.Storage{DB: db})
	ctx := context.Background()
	acc1 := m.createTestAccount(ctx, t, "", nil)
	acc2 := m.createTestAccount(ctx, t, "", map[string]interface{}{"one": "foo", "two": "bar"})
	acp1 := m.createTestControlProgram(ctx, t, acc1.ID)
	acp2 := m.createTestControlProgram(ctx, t, acc2.ID)

	txs := []map[string]interface{}{{
		"inputs": []interface{}{},
		"outputs": []interface{}{
			map[string]interface{}{
				"control_program": hex.EncodeToString(acp1),
			},
			map[string]interface{}{
				"control_program": hex.EncodeToString(acp2),
			},
		},
	}}

	wantTags := []byte(`{"one": "foo", "two": "bar"}`)

	want := []map[string]interface{}{{
		"inputs": []interface{}{},
		"outputs": []interface{}{
			map[string]interface{}{
				"purpose":         "receive",
				"control_program": hex.EncodeToString(acp1),
				"account_id":      acc1.ID,
			},
			map[string]interface{}{
				"purpose":         "receive",
				"control_program": hex.EncodeToString(acp2),
				"account_id":      acc2.ID,
				"account_tags":    (*json.RawMessage)(&wantTags),
			},
		},
	}}

	err := m.AnnotateTxs(ctx, txs, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(txs, want) {
		t.Errorf("AnnotateTxs = %+v want %+v", txs, want)
	}
}
