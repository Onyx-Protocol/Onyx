package txdb

import (
	"testing"

	"chain/database/pg/pgtest"
	"chain/fedchain/bc"
	"chain/testutil"
)

func TestAddIssuances(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	aid := [32]byte{255}

	// creates new record if it doesn't exist
	err := addIssuances(ctx, map[bc.AssetID]uint64{
		aid: 10,
	}, true)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	conf, total, err := issued(ctx, aid)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, conf, uint64(10), "confirmed issued")
	testutil.ExpectEqual(t, total, uint64(10), "total issued")

	// updates existing record if it does
	err = addIssuances(ctx, map[bc.AssetID]uint64{
		aid: 5,
	}, true)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	conf, total, err = issued(ctx, aid)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, conf, uint64(15), "confirmed issued")
	testutil.ExpectEqual(t, total, uint64(15), "total issued")

	// updates just pool
	err = addIssuances(ctx, map[bc.AssetID]uint64{
		aid: 5,
	}, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	conf, total, err = issued(ctx, aid)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	testutil.ExpectEqual(t, conf, uint64(15), "confirmed issued")
	testutil.ExpectEqual(t, total, uint64(20), "total issued")
}

func TestSetIssuances(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	aid := [32]byte{255}

	err := addIssuances(ctx, map[bc.AssetID]uint64{aid: 10}, true)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = addIssuances(ctx, map[bc.AssetID]uint64{aid: 10}, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = setIssuances(ctx, map[bc.AssetID]uint64{aid: 6})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	conf, total, err := issued(ctx, aid)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	testutil.ExpectEqual(t, conf, uint64(10), "confirmed issued")
	testutil.ExpectEqual(t, total, uint64(16), "total issued")
}
