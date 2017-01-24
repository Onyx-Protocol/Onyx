package asset

import (
	"context"
	"testing"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

const rawdef = `{
  "currency": "USD"
}`

type fakeSaver func(context.Context, bc.AssetID, map[string]interface{}, string) error

func (f fakeSaver) SaveAnnotatedAsset(ctx context.Context, assetID bc.AssetID, obj map[string]interface{}, sortID string) error {
	return f(ctx, assetID, obj, sortID)
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
							TypedInput: &bc.IssuanceInput{
								Amount: 10000,
								IssuanceWitness: bc.IssuanceWitness{
									InitialBlock:    r.initialBlockHash,
									AssetDefinition: []byte(rawdef),
									IssuanceProgram: issuanceProgram,
									VMVersion:       remotevmver,
								},
							},
						},
						{ // local asset
							AssetVersion: 1,
							TypedInput: &bc.IssuanceInput{
								Amount: 10000,
								IssuanceWitness: bc.IssuanceWitness{
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
	remoteAssetID := b.Transactions[0].Inputs[0].AssetID()

	var assetsSaved []bc.AssetID
	r.indexer = fakeSaver(func(ctx context.Context, assetID bc.AssetID, obj map[string]interface{}, sortID string) error {
		assetsSaved = append(assetsSaved, assetID)
		return nil
	})

	// Call the block callback and index the remote asset.
	err = r.indexAssets(ctx, b)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the annotated asset got saved to the query indexer.
	if !testutil.DeepEqual(assetsSaved, []bc.AssetID{remoteAssetID}) {
		t.Errorf("saved annotated assets got %#v, want %#v", assetsSaved, []bc.AssetID{remoteAssetID})
	}

	// Ensure that the asset was saved to the `assets` table.
	got, err := r.findByID(ctx, remoteAssetID)
	if err != nil {
		t.Fatal(err)
	}
	want := &Asset{
		AssetID:          remoteAssetID,
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
