package generator_test

import (
	"testing"
	"time"

	"chain/api/asset/assettest"
	. "chain/api/generator"
	"chain/database/pg/pgtest"
	"chain/fedchain/txscript"
	"chain/testutil"
)

func TestGetAndAddBlockSignatures(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	fc, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, prev, err := fc.GenerateBlock(ctx, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = GetAndAddBlockSignatures(ctx, block, prev)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	engine, err := txscript.NewEngineForBlock(ctx, prev.OutputScript, block, txscript.StandardVerifyFlags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = engine.Execute()
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
