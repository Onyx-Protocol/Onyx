package voting

import (
	"testing"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/generator"
	"chain/core/issuer"
	"chain/core/txbuilder"
	"chain/cos/bc"
)

func createVotingTokenFixture(ctx context.Context, t *testing.T, g *generator.Generator, right bc.AssetID, admin []byte, amount uint64) *Token {
	assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")

	assetAmount := bc.AssetAmount{AssetID: assetID, Amount: amount}
	issueTxTemplate, err := issuer.Issue(ctx, assetAmount, []*txbuilder.Destination{
		{
			AssetAmount: assetAmount,
			Receiver:    TokenIssuance(ctx, right, admin),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := asset.FinalizeTx(ctx, issueTxTemplate)
	if err != nil {
		t.Fatal(err)
	}
	// Confirm it in a block
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	token, err := FindTokenForOutpoint(ctx, bc.Outpoint{Hash: tx.Hash, Index: 0})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func createVotingRightFixture(ctx context.Context, t *testing.T, g *generator.Generator, holder []byte) *Right {
	assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")

	assetAmount := bc.AssetAmount{AssetID: assetID, Amount: 1}
	issueTxTemplate, err := issuer.Issue(ctx, assetAmount, []*txbuilder.Destination{
		{
			AssetAmount: assetAmount,
			Receiver:    RightIssuance(ctx, holder, holder),
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
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	right, err := GetCurrentHolder(ctx, assetID)
	if err != nil {
		t.Fatal(err)
	}
	return right
}
