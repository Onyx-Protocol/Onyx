package query

import (
	"context"
	"testing"
	"unicode"

	"chain/database/pg/pgtest"
	"chain/protocol/bc/bctest"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
)

func TestAnnotatedTxs(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	c := prottest.NewChain(t)
	indexer := NewIndexer(db, c, nil)
	b := &legacy.Block{
		Transactions: []*legacy.Tx{
			bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash()),
			bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash()),
		},
	}
	txs, err := indexer.insertAnnotatedTxs(ctx, b)
	if err != nil {
		t.Error(err)
	}
	if len(txs) != len(b.Transactions) {
		t.Errorf("Got %d transactions, expected %d", len(txs), len(b.Transactions))
	}
}

func TestAnnotatedTxsReferenceData(t *testing.T) {
	ctx := context.Background()

	referenceData := []string{
		"",
		"{\"\u0000\": \"world\"}",
		"{\"badString\":\"\u0000\"}",
		`{"badString": "\u0000"}`,
		`{"🤦‍♀️": "🤦‍♀️"}`,
		`🤦‍♀️`,
		"{",
		"\u0000",
		"\u0001",
		"{\"client_id\": 1, \"device_name\": \"FooBar\ufffd\u0000\ufffd\u000f\ufffd\"}",
		`{"client_id": 1, "device_name": "FooBar\ufffd\u0000\ufffd\u000f\ufffd"}`,
		string(unicode.MaxRune + 1),
		`"` + string(unicode.MaxRune+1) + `"`,
		string([]byte{0xff, 0xfe, 0xfd}),
		`"` + string([]byte{0xff, 0xfe, 0xfd}) + `"`,
	}
	for _, refData := range referenceData {
		t.Run(refData, func(t *testing.T) {
			db := pgtest.NewTx(t)
			c := prottest.NewChain(t)
			indexer := NewIndexer(db, c, nil)

			setRefData := func(tx *legacy.Tx) { tx.ReferenceData = []byte(refData) }
			b := &legacy.Block{
				Transactions: []*legacy.Tx{
					bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash(), setRefData),
				},
			}
			txs, err := indexer.insertAnnotatedTxs(ctx, b)
			if err != nil {
				t.Error(err)
			}
			if len(txs) != len(b.Transactions) {
				t.Errorf("Got %d transactions, expected %d", len(txs), len(b.Transactions))
			}
		})
	}
}
