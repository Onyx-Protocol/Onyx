package asset

import (
	"context"
	"reflect"
	"testing"

	"chain-stealth/core/confidentiality"
	"chain-stealth/crypto/ca"
	"chain-stealth/crypto/ed25519"
	"chain-stealth/database/pg/pgtest"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/prottest"
	"chain-stealth/testutil"
)

const def = `{"currency":"USD"}`

type fakeSaver func(context.Context, bc.AssetID, map[string]interface{}, string) error

func (f fakeSaver) SaveAnnotatedAsset(ctx context.Context, assetID bc.AssetID, obj map[string]interface{}, sortID string) error {
	return f(ctx, assetID, obj, sortID)
}

func TestIndexNonLocalAssets(t *testing.T) {
	db := pgtest.NewTx(t)
	r := NewRegistry(db, prottest.NewChain(t), nil, &confidentiality.Storage{DB: db})
	ctx := context.Background()

	// Create a local asset which should be unaffected by a block landing.
	local, err := r.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create the issuance program of a remote asset.
	issuanceProgram, err := programWithDefinition([]ed25519.PublicKey{testutil.TestPub}, 1, []byte(def))
	if err != nil {
		t.Fatal(err)
	}

	txin, _, err := bc.NewConfidentialIssuanceInput([]byte{0x01}, 10000, nil, local.InitialBlockHash, local.IssuanceProgram, nil, ca.RecordKey{0x01})
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
							TypedInput: &bc.IssuanceInput1{
								Amount: 10000,
								AssetWitness: bc.AssetWitness{
									InitialBlock:    r.initialBlockHash,
									IssuanceProgram: issuanceProgram,
									VMVersion:       1,
								},
							},
						},
						txin, // local asset
					},
				},
			},
		},
	}
	remoteAssetID, _ := b.Transactions[0].Inputs[0].AssetID()

	var assetsSaved []bc.AssetID
	r.indexer = fakeSaver(func(ctx context.Context, assetID bc.AssetID, obj map[string]interface{}, sortID string) error {
		assetsSaved = append(assetsSaved, assetID)
		return nil
	})

	// Call the block callback and index the remote asset.
	r.indexAssets(ctx, b)

	// Ensure that the annotated asset got saved to the query indexer.
	if !reflect.DeepEqual(assetsSaved, []bc.AssetID{remoteAssetID}) {
		t.Errorf("saved annotated assets got %#v, want %#v", assetsSaved, []bc.AssetID{remoteAssetID})
	}

	// Ensure that the asset was saved to the `assets` table.
	got, err := r.findByID(ctx, remoteAssetID)
	if err != nil {
		t.Fatal(err)
	}
	want := &Asset{
		AssetID: remoteAssetID,
		Definition: map[string]interface{}{
			"currency": "USD",
		},
		IssuanceProgram:  issuanceProgram,
		InitialBlockHash: r.initialBlockHash,
		sortID:           got.sortID,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lookupAsset() = %#v, want %#v", got, want)
	}
}
