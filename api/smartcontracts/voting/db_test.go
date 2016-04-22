package voting

import (
	"os"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset/assettest"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "votingsystemtest", "../../appdb/schema.sql")
}

// TestInsertVotingRightAccountID tests inserting a voting right into the
// database with a holder script that is the address of an account. The
// voting_rights_txs row should contain the correct account id.
func TestInsertVotingRightAccountID(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	var (
		accountID = assettest.CreateAccountFixture(ctx, t, "", "", nil)
		assetID   = assettest.CreateAssetFixture(ctx, t, "", "", "")
		address   = assettest.CreateAddressFixture(ctx, t, accountID)
	)

	data := rightScriptData{
		HolderScript:   address.PKScript,
		OwnershipChain: bc.Hash{},
		Deadline:       1458172911,
		Delegatable:    true,
	}

	err := insertVotingRight(ctx, assetID, 1, 0, bc.Outpoint{}, data)
	if err != nil {
		t.Fatal(err)
	}

	// Look up the inserted voting right.
	var dbAccountID string
	err = pg.QueryRow(ctx, "SELECT account_id FROM voting_right_txs WHERE tx_hash = $1 AND index = $2", bc.Hash{}, 0).
		Scan(&dbAccountID)
	if err != nil {
		t.Fatal(err)
	}

	// The voting_right_txs row should have the correct account ID.
	if accountID != dbAccountID {
		t.Errorf("got=%s, want=%s", dbAccountID, accountID)
	}
}

// TestUpsertVotingToken tests inserting, updating and retrieving a voting
// token from the database index.
func TestUpsertVotingToken(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	var (
		tokenAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		rightAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
		out1         = bc.Outpoint{Hash: exampleHash, Index: 6}
		out2         = bc.Outpoint{Hash: exampleHash2, Index: 22}
	)
	data := tokenScriptData{
		Right:       rightAssetID,
		AdminScript: []byte{0x01, 0x02, 0x03},
		OptionCount: 10,
		State:       stateDistributed,
		SecretHash:  bc.Hash{},
		Vote:        0,
	}

	err := upsertVotingToken(ctx, tokenAssetID, out1, 100, data)
	if err != nil {
		t.Fatal(err)
	}

	// Modify the token state, and upsert it again.
	data.State, data.Vote = stateVoted, 2
	err = upsertVotingToken(ctx, tokenAssetID, out2, 100, data)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch the token from the db.
	tok, err := FindTokenForAsset(ctx, tokenAssetID, rightAssetID)
	if err != nil {
		t.Fatal(err)
	}
	want := &Token{
		AssetID:         tokenAssetID,
		Outpoint:        out2,
		Amount:          100,
		tokenScriptData: data,
	}
	if !reflect.DeepEqual(tok, want) {
		t.Errorf("got=%#v want=%#v", tok, want)
	}
}

func TestTallyVotes(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	type testVoteToken struct {
		state  TokenState
		amount uint64
		vote   int64
	}

	testCases := []struct {
		options int64
		seed    []testVoteToken
		want    Tally
	}{
		{
			// All tokens in distributed state.
			options: 2,
			seed: []testVoteToken{
				{state: stateDistributed, amount: 100, vote: 0},
				{state: stateDistributed, amount: 300, vote: 0},
				{state: stateDistributed, amount: 600, vote: 0},
			},
			want: Tally{
				Circulation: 1000,
				Distributed: 1000,
				Votes:       []int{0, 0},
			},
		},
		{
			// All tokens distributed or intended.
			options: 5,
			seed: []testVoteToken{
				{state: stateDistributed, amount: 100, vote: 0},
				{state: stateIntended, amount: 200, vote: 0},
				{state: stateIntended, amount: 10, vote: 0},
				{state: stateDistributed, amount: 300, vote: 0},
				{state: stateDistributed, amount: 500, vote: 0},
				{state: stateIntended, amount: 1, vote: 0},
			},
			want: Tally{
				Circulation: 1111,
				Distributed: 900,
				Intended:    211,
				Votes:       []int{0, 0, 0, 0, 0},
			},
		},
		{
			// Mix of voted, distributed and intended.
			options: 2,
			seed: []testVoteToken{
				{state: stateDistributed, amount: 100, vote: 0},
				{state: stateVoted, amount: 200, vote: 1},
				{state: stateIntended, amount: 10, vote: 0},
				{state: stateDistributed, amount: 300, vote: 0},
				{state: stateVoted, amount: 500, vote: 2},
				{state: stateVoted, amount: 1, vote: 1},
			},
			want: Tally{
				Circulation: 1111,
				Distributed: 400,
				Intended:    10,
				Voted:       701,
				Votes:       []int{201, 500},
			},
		},
		{
			// Non-zero votes for tokens in states besides `stateVoted` should
			// not be tallied.
			options: 2,
			seed: []testVoteToken{
				{state: stateDistributed, amount: 499, vote: 2},
				{state: stateVoted, amount: 1, vote: 1},
			},
			want: Tally{
				Circulation: 500,
				Distributed: 499,
				Voted:       1,
				Votes:       []int{1, 0},
			},
		},
		{
			// Closed vote
			options: 2,
			seed: []testVoteToken{
				{state: stateVoted | stateFinished, amount: 100, vote: 1},
				{state: stateDistributed | stateFinished, amount: 10, vote: 0},
				{state: stateVoted | stateFinished, amount: 10000, vote: 2},
				{state: stateIntended | stateFinished, amount: 3, vote: 0},
				{state: stateVoted | stateFinished, amount: 1000, vote: 1},
				{state: stateVoted | stateFinished, amount: 100, vote: 2},
			},
			want: Tally{
				Circulation: 11213,
				Distributed: 10,
				Intended:    3,
				Voted:       11200,
				Closed:      11213,
				Votes:       []int{1100, 10100},
			},
		},
	}

	for i, tc := range testCases {
		assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")

		for j, vt := range tc.seed {
			rightAssetID := assettest.CreateAssetFixture(ctx, t, "", "", "")
			err := upsertVotingToken(ctx, assetID, bc.Outpoint{Index: uint32(i)}, vt.amount, tokenScriptData{
				Right:       rightAssetID,
				AdminScript: []byte{txscript.OP_1},
				OptionCount: tc.options,
				State:       vt.state,
				Vote:        vt.vote,
			})
			if err != nil {
				t.Fatalf("Error setting up test case %d, seed %d", i, j)
			}
		}

		got, err := TallyVotes(ctx, assetID)
		if err != nil {
			t.Fatalf("%d: error tallying votes: %s", i, err)
		}
		tc.want.AssetID = assetID
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("test case %d:\ngot=%#v\nwant=%#v", i, got, tc.want)
		}
	}
}
