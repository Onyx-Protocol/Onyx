package asset

import (
	"context"
	"testing"

	"chain/core/query"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
	"chain/testutil"
)

const (
	rawdef = `{
  "currency": "USD"
}`
	notJSON = `{{{{{{{`
)

type fakeSaver func(context.Context, *query.AnnotatedAsset, string) error

func (f fakeSaver) SaveAnnotatedAsset(ctx context.Context, aa *query.AnnotatedAsset, sortID string) error {
	return f(ctx, aa, sortID)
}

func TestIndexNonLocalAssets(t *testing.T) {
	r := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()

	// Create a local asset which should be unaffected by a block landing.
	local, err := r.Define(ctx, []chainkd.XPub{testutil.TestXPub}, 1, nil, "", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	localdef := local.RawDefinition()

	// Create the issuance program of a remote asset.
	issuanceProgram, remotevmver, err := multisigIssuanceProgram([]ed25519.PublicKey{testutil.TestPub, testutil.TestPub}, 2)
	if err != nil {
		t.Fatal(err)
	}
	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Height: 22,
		},
		Transactions: []*legacy.Tx{
			{
				TxData: legacy.TxData{
					Inputs: []*legacy.TxInput{
						{ // non-local asset
							AssetVersion: 1,
							TypedInput: &legacy.IssuanceInput{
								Amount: 10000,
								IssuanceWitness: legacy.IssuanceWitness{
									InitialBlock:    r.initialBlockHash,
									AssetDefinition: []byte(rawdef),
									IssuanceProgram: issuanceProgram,
									VMVersion:       remotevmver,
								},
							},
						},
						{ // non-local asset, non-JSON asset definition
							AssetVersion: 1,
							TypedInput: &legacy.IssuanceInput{
								Amount: 10000,
								IssuanceWitness: legacy.IssuanceWitness{
									InitialBlock:    r.initialBlockHash,
									AssetDefinition: []byte(notJSON),
									IssuanceProgram: issuanceProgram,
									VMVersion:       remotevmver,
								},
							},
						},
						{ // local asset
							AssetVersion: 1,
							TypedInput: &legacy.IssuanceInput{
								Amount: 10000,
								IssuanceWitness: legacy.IssuanceWitness{
									InitialBlock:    r.initialBlockHash,
									AssetDefinition: localdef,
									IssuanceProgram: local.IssuanceProgram,
									VMVersion:       local.VMVersion,
								},
							},
						},
					},
				},
			},
		},
	}
	remoteAssetID1 := b.Transactions[0].Inputs[0].AssetID()
	remoteAssetID2 := b.Transactions[0].Inputs[1].AssetID()

	assetsSaved := make(map[bc.AssetID]bool)
	r.indexer = fakeSaver(func(ctx context.Context, aa *query.AnnotatedAsset, sortID string) error {
		assetsSaved[aa.ID] = true
		return nil
	})

	// Call the block callback and index the remote assets.
	err = r.indexAssets(ctx, b)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the annotated asset got saved to the query indexer.
	if !testutil.DeepEqual(assetsSaved, map[bc.AssetID]bool{remoteAssetID1: true, remoteAssetID2: true}) {
		t.Errorf("saved annotated assets got %#v, want %#v", assetsSaved, []bc.AssetID{remoteAssetID1, remoteAssetID2})
	}

	// Ensure that the assets were saved to the `assets` table.
	got, err := r.findByID(ctx, remoteAssetID1)
	if err != nil {
		t.Fatal(err)
	}
	want := &Asset{
		AssetID:          remoteAssetID1,
		VMVersion:        remotevmver,
		IssuanceProgram:  issuanceProgram,
		InitialBlockHash: r.initialBlockHash,
		sortID:           got.sortID,
	}
	err = want.SetDefinition(map[string]interface{}{
		"currency": "USD",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !testutil.DeepEqual(got, want) {
		t.Errorf("lookupAsset() = %#v, want %#v", got, want)
	}
}
