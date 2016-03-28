package txdb

import (
	"testing"

	"chain/database/pg/pgtest"
	"chain/fedchain/bc"
	"chain/testutil"
)

func TestAddIssuances(t *testing.T) {
	cases := []struct {
		issuedAmt1, destroyedAmt1 uint64
		issuedAmt2, destroyedAmt2 uint64
		conf1, conf2              bool
		confirmed, total          uint64
	}{
		{5, 0, 5, 0, true, true, 10, 10},
		{5, 0, 5, 0, true, false, 5, 10},
		{5, 0, 5, 0, false, true, 5, 10},
		{5, 0, 5, 0, false, false, 0, 10},

		{5, 1, 5, 1, true, true, 8, 8},
		{5, 1, 5, 1, true, false, 4, 8},
		{5, 1, 5, 1, false, true, 4, 8},
		{5, 1, 5, 1, false, false, 0, 8},

		{5, 0, 0, 1, true, true, 4, 4},
		{5, 0, 0, 1, true, false, 5, 4},
		{5, 0, 0, 1, false, false, 0, 4},

		{0, 1, 5, 0, true, true, 4, 4},
		{0, 1, 5, 0, false, true, 5, 4},
		{0, 1, 5, 0, false, false, 0, 4},
	}
	aid := [32]byte{255}

	for _, c := range cases {
		ctx := pgtest.NewContext(t)
		defer pgtest.Finish(ctx)

		err := addIssuances(ctx, amtMap(aid, c.issuedAmt1), amtMap(aid, c.destroyedAmt1), c.conf1)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		err = addIssuances(ctx, amtMap(aid, c.issuedAmt2), amtMap(aid, c.destroyedAmt2), c.conf2)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		gotConf, gotTotal, err := circulation(ctx, aid)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		testutil.ExpectEqual(t, gotConf, c.confirmed, "confirmed issued")
		testutil.ExpectEqual(t, gotTotal, c.total, "total issued")

		pgtest.Finish(ctx)
	}
}

func amtMap(aid bc.AssetID, amt uint64) map[bc.AssetID]uint64 {
	if amt == 0 {
		return nil
	}
	return map[bc.AssetID]uint64{aid: amt}
}

func TestSetIssuances(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	aid := [32]byte{255}

	err := addIssuances(ctx, map[bc.AssetID]uint64{aid: 10}, nil, true)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = addIssuances(ctx, map[bc.AssetID]uint64{aid: 10}, map[bc.AssetID]uint64{aid: 5}, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = setIssuances(ctx, map[bc.AssetID]uint64{aid: 8}, map[bc.AssetID]uint64{aid: 2})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	conf, total, err := circulation(ctx, aid)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	testutil.ExpectEqual(t, conf, uint64(10), "confirmed issued")
	testutil.ExpectEqual(t, total, uint64(16), "total issued")
}
