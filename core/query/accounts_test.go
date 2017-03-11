package query

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestQueryAccounts(t *testing.T) {
	ctx := context.Background()
	indexer := NewIndexer(pgtest.NewTx(t), prottest.NewChain(t), nil)

	// Save a bunch of annotated accounts to the database.
	seedAccounts := map[string]*AnnotatedAccount{
		"accAlice": {
			ID:    "accAlice",
			Alias: "alice",
			Keys: []*AccountKey{
				{RootXPub: chainkd.XPub{1}, AccountXPub: chainkd.XPub{2}},
			},
			Quorum: 1,
			Tags:   raw(`{"branch_id": "NYC1", "internal_account_id": "alice123"}`),
		},
		"accBob": {
			ID:    "accBob",
			Alias: "bob",
			Keys: []*AccountKey{
				{RootXPub: chainkd.XPub{1}, AccountXPub: chainkd.XPub{3}},
			},
			Quorum: 1,
			Tags:   raw(`{"branch_id": "SFO1", "internal_account_id": "bobbie"}`),
		},
		"accCarol": {
			ID:    "accCarol",
			Alias: "carol",
			Keys: []*AccountKey{
				{RootXPub: chainkd.XPub{1}, AccountXPub: chainkd.XPub{4}},
				{RootXPub: chainkd.XPub{1}, AccountXPub: chainkd.XPub{5}},
			},
			Quorum: 2,
			Tags:   raw(`{"branch_id": "NYC1", "internal_account_id": "carolmerryl"}`),
		},
	}
	for _, acc := range seedAccounts {
		err := indexer.SaveAnnotatedAccount(ctx, acc)
		if err != nil {
			testutil.FatalErr(t, err)
		}
	}

	testCases := []struct {
		filt    string
		vals    []interface{}
		wantErr error
		want    []*AnnotatedAccount
	}{
		{
			filt:    "alias = $1",
			vals:    []interface{}{}, // mismatch
			wantErr: ErrParameterCountMismatch,
		},
		{
			filt: "alias = 'alice'",
			want: []*AnnotatedAccount{seedAccounts["accAlice"]},
		},
		{
			filt: "id = 'accAlice'",
			want: []*AnnotatedAccount{seedAccounts["accAlice"]},
		},
		{
			filt: "alias = $1",
			vals: []interface{}{"bob"},
			want: []*AnnotatedAccount{seedAccounts["accBob"]},
		},
		{
			filt: "quorum = 1",
			want: []*AnnotatedAccount{
				seedAccounts["accBob"],
				seedAccounts["accAlice"],
			},
		},
		{
			filt: "tags.branch_id = 'NYC1'",
			want: []*AnnotatedAccount{
				seedAccounts["accCarol"],
				seedAccounts["accAlice"],
			},
		},
	}
	for _, tc := range testCases {
		accs, _, err := indexer.Accounts(ctx, tc.filt, tc.vals, "", 100)
		if !testutil.DeepEqual(err, tc.wantErr) {
			t.Errorf("%q got error %#v, want error %#v", tc.filt, err, tc.wantErr)
		}
		if !testutil.DeepEqual(accs, tc.want) {
			t.Errorf("%q got %s, want %s", tc.filt, spew.Sdump(accs), spew.Sdump(tc.want))
		}
	}
}

func raw(s string) *json.RawMessage {
	rawMsg := json.RawMessage(s)
	return &rawMsg
}
