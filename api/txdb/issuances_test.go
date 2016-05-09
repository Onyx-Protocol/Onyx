package txdb

import (
	"testing"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
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
		ctx := context.Background()
		dbtx := pgtest.NewTx(t)

		err := addIssuances(ctx, dbtx, map[bc.AssetID]*state.AssetState{
			aid: &state.AssetState{Issuance: c.issuedAmt1, Destroyed: c.destroyedAmt1},
		}, c.conf1)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		err = addIssuances(ctx, dbtx, map[bc.AssetID]*state.AssetState{
			aid: &state.AssetState{Issuance: c.issuedAmt2, Destroyed: c.destroyedAmt2},
		}, c.conf2)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		gotConf, gotTotal, err := circulationForTest(ctx, dbtx, aid)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		testutil.ExpectEqual(t, gotConf, c.confirmed, "confirmed issued")
		testutil.ExpectEqual(t, gotTotal, c.total, "total issued")
	}
}

func TestSetIssuances(t *testing.T) {
	ctx := context.Background()
	dbtx := pgtest.NewTx(t)

	aid := [32]byte{255}

	err := addIssuances(ctx, dbtx, map[bc.AssetID]*state.AssetState{
		aid: &state.AssetState{Issuance: 10},
	}, true)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = addIssuances(ctx, dbtx, map[bc.AssetID]*state.AssetState{
		aid: &state.AssetState{Issuance: 10, Destroyed: 5},
	}, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = setIssuances(ctx, dbtx, map[bc.AssetID]*state.AssetState{
		aid: &state.AssetState{Issuance: 8, Destroyed: 2},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	conf, total, err := circulationForTest(ctx, dbtx, aid)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	testutil.ExpectEqual(t, conf, uint64(10), "confirmed issued")
	testutil.ExpectEqual(t, total, uint64(16), "total issued")
}

// circulationForTest returns the confirmed and total
// circulationForTest amounts for the given asset.
func circulationForTest(ctx context.Context, dbtx *sql.Tx, assetID bc.AssetID) (confirmed, total uint64, err error) {
	const q = `
		SELECT (confirmed - destroyed_confirmed),
		(confirmed + pool - destroyed_confirmed - destroyed_pool)
		FROM issuance_totals WHERE asset_id=$1
	`
	err = dbtx.QueryRow(ctx, q, assetID).Scan(&confirmed, &total)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	} else if err != nil {
		return 0, 0, errors.Wrap(err, "loading issued and destroyed amounts")
	}
	return confirmed, total, nil
}
