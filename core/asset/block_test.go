package asset

import (
	"context"
	"reflect"
	"testing"

	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

const def = `{"currency":"USD"}`

type fakeSaver func(context.Context, bc.AssetID, map[string]interface{}, string) error

func (f fakeSaver) SaveAnnotatedAsset(ctx context.Context, assetID bc.AssetID, obj map[string]interface{}, sortID string) error {
	return f(ctx, assetID, obj, sortID)
}

func TestIndexNonLocalAssets(t *testing.T) {
	db := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), db)
	Init(prottest.NewChain(t), nil)

	initial, err := chain.GetBlock(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Create a local asset which should be unaffected by a block landing.
	local, err := Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, initial.Hash(), "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create the issuance program of a remote asset.
	issuanceProgram, err := programWithDefinition([]ed25519.PublicKey{testutil.TestPub}, 1, []byte(def))
	if err != nil {
		t.Fatal(err)
	}
	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Height: 22,
		},
		Transactions: []*bc.Tx{
			{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{ // non-local asset
							AssetVersion: 1,
							InputCommitment: &bc.IssuanceInputCommitment{
								InitialBlock:    initial.Hash(),
								Amount:          10000,
								IssuanceProgram: issuanceProgram,
								VMVersion:       1,
							},
						},
						{ // local asset
							AssetVersion: 1,
							InputCommitment: &bc.IssuanceInputCommitment{
								InitialBlock:    initial.Hash(),
								Amount:          10000,
								IssuanceProgram: local.IssuanceProgram,
								VMVersion:       1,
							},
						},
					},
				},
			},
		},
	}
	remoteAssetID := b.Transactions[0].Inputs[0].AssetID()

	var assetsSaved []bc.AssetID
	indexer = fakeSaver(func(ctx context.Context, assetID bc.AssetID, obj map[string]interface{}, sortID string) error {
		assetsSaved = append(assetsSaved, assetID)
		return nil
	})

	// Call the block callback and index the remote asset.
	indexAssets(ctx, b)

	// Ensure that the annotated asset got saved to the query indexer.
	if !reflect.DeepEqual(assetsSaved, []bc.AssetID{remoteAssetID}) {
		t.Errorf("saved annotated assets got %#v, want %#v", assetsSaved, []bc.AssetID{remoteAssetID})
	}

	// Ensure that the asset was saved to the `assets` table.
	got, err := lookupAsset(ctx, remoteAssetID, "")
	if err != nil {
		t.Fatal(err)
	}
	want := &Asset{
		AssetID: remoteAssetID,
		Definition: map[string]interface{}{
			"currency": "USD",
		},
		IssuanceProgram:  issuanceProgram,
		InitialBlockHash: initial.Hash(),
		sortID:           got.sortID,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lookupAsset() = %#v, want %#v", got, want)
	}
}
