package voting

import (
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/api/issuer"
	"chain/api/txbuilder"
	"chain/cos/bc"
	"chain/crypto/hash256"
)

func createVotingTokenFixture(ctx context.Context, t *testing.T, right bc.AssetID, admin []byte, amount uint64) *Token {
	assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")

	issueTxTemplate, err := issuer.Issue(ctx, assetID, []*txbuilder.Destination{
		{
			AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: amount},
			Receiver:    TokenIssuance(ctx, right, admin, 2, hash256.Sum(right[:])),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = asset.FinalizeTx(ctx, issueTxTemplate)
	if err != nil {
		t.Fatal(err)
	}
	// Confirm it in a block
	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	token, err := FindTokenForAsset(ctx, assetID, right)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func createVotingRightFixture(ctx context.Context, t *testing.T, holder []byte) *RightWithUTXO {
	assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")

	issueTxTemplate, err := issuer.Issue(ctx, assetID, []*txbuilder.Destination{
		{
			AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 1},
			Receiver:    RightIssuance(ctx, holder, holder),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	holdingTx, err := asset.FinalizeTx(ctx, issueTxTemplate)
	if err != nil {
		t.Fatal(err)
	}
	// Confirm it in a block
	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	right, err := FindRightForOutpoint(ctx, bc.Outpoint{Hash: holdingTx.Hash, Index: 0})
	if err != nil {
		t.Fatal(err)
	}
	return right
}
