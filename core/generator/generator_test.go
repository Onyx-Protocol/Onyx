package generator

import (
	"context"
	"testing"
	"time"

	"chain/core/blocksigner"
	"chain/core/mockhsm"
	"chain/cos"
	"chain/cos/mempool"
	"chain/cos/memstore"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/crypto/ed25519"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestGetAndAddBlockSignatures(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	fc, err := cos.NewFC(ctx, memstore.New(), mempool.New(), nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// TODO(kr): tweak the generator's design to not
	// take a hard dependency on mockhsm.
	// See also similar comment in $CHAIN/core/blocksigner/blocksigner.go.
	hsm := mockhsm.New(dbtx)
	xpub, err := hsm.CreateKey(ctx, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	localSigner := blocksigner.New(xpub.XPub, hsm, dbtx, fc)
	config := Config{
		LocalSigner:  localSigner,
		BlockPeriod:  time.Second,
		BlockKeys:    []ed25519.PublicKey{xpub.Key},
		SigsRequired: 1,
		FC:           fc,
	}
	genesis, err := config.UpsertGenesisBlock(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	g := New(genesis, state.Empty(), config)

	tip, snapshot, err := fc.Recover(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, err := fc.GenerateBlock(ctx, tip, snapshot, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = g.GetAndAddBlockSignatures(ctx, block, tip)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	engine, err := txscript.NewEngineForBlock(tip.ConsensusProgram, block, txscript.StandardVerifyFlags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = engine.Execute()
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
