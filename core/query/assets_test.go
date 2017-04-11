package query

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestQueryAssets(t *testing.T) {
	ctx := context.Background()
	indexer := NewIndexer(pgtest.NewTx(t), prottest.NewChain(t), nil)

	// Save a bunch of annotated assets to the database.
	seedAssets := map[string]*AnnotatedAsset{
		"asset1": {
			ID:              bc.NewAssetID([32]byte{1}),
			Alias:           "dollars",
			IssuanceProgram: []byte{0xde, 0xad, 0xbe, 0xef},
			Keys: []*AssetKey{
				{RootXPub: chainkd.XPub{1}, AssetPubkey: []byte{0x02}},
			},
			Quorum:     1,
			Definition: raw(`{"currency_code": "USD"}`),
			Tags:       raw(`{"grade": "A"}`),
			IsLocal:    true,
		},
		"asset2": {
			ID:              bc.NewAssetID([32]byte{2}),
			Alias:           "gold",
			IssuanceProgram: []byte{0xde, 0xad, 0xbe, 0xef},
			Keys: []*AssetKey{
				{RootXPub: chainkd.XPub{1}, AssetPubkey: []byte{0x03}},
				{RootXPub: chainkd.XPub{1}, AssetPubkey: []byte{0x04}},
			},
			Quorum:     2,
			Definition: raw(`{}`),
			Tags:       raw(`{"grade": "A"}`),
			IsLocal:    true,
		},
		"asset3": {
			ID:              bc.NewAssetID([32]byte{3}),
			IssuanceProgram: []byte{0xc0, 0x01, 0xca, 0xfe},
			Keys: []*AssetKey{
				{RootXPub: chainkd.XPub{1}, AssetPubkey: []byte{0x05}},
			},
			Quorum:     1,
			Definition: raw(`{}`),
			Tags:       raw(`{}`),
			IsLocal:    false,
		},
	}
	for sortID, asset := range seedAssets {
		err := indexer.SaveAnnotatedAsset(ctx, asset, sortID)
		if err != nil {
			testutil.FatalErr(t, err)
		}
	}

	testCases := []struct {
		filt    string
		vals    []interface{}
		wantErr error
		want    []*AnnotatedAsset
	}{
		{
			filt:    "alias = $1",
			vals:    []interface{}{}, // mismatch
			wantErr: ErrParameterCountMismatch,
		},
		{
			filt: "alias = 'dollars'",
			want: []*AnnotatedAsset{seedAssets["asset1"]},
		},
		{
			filt: "is_local = 'no'",
			want: []*AnnotatedAsset{seedAssets["asset3"]},
		},
		{
			filt: "is_local = 'yes'",
			want: []*AnnotatedAsset{seedAssets["asset2"], seedAssets["asset1"]},
		},
		{
			filt: "tags.grade = $1",
			vals: []interface{}{"A"},
			want: []*AnnotatedAsset{seedAssets["asset2"], seedAssets["asset1"]},
		},
		{
			filt: "definition.currency_code = 'USD'",
			want: []*AnnotatedAsset{seedAssets["asset1"]},
		},
		{
			filt: "quorum = 2",
			want: []*AnnotatedAsset{seedAssets["asset2"]},
		},
		{
			filt: "issuance_program = 'c001cafe'",
			want: []*AnnotatedAsset{seedAssets["asset3"]},
		},
	}
	for _, tc := range testCases {
		accs, _, err := indexer.Assets(ctx, tc.filt, tc.vals, "", 100)
		if !testutil.DeepEqual(err, tc.wantErr) {
			t.Errorf("%q got error %#v, want error %#v", tc.filt, err, tc.wantErr)
		}
		if !testutil.DeepEqual(accs, tc.want) {
			t.Errorf("%q got %s, want %s", tc.filt, spew.Sdump(accs), spew.Sdump(tc.want))
		}
	}
}
