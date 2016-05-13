package voting

import (
	"fmt"
	"reflect"
	"testing"

	"chain/core/asset/assettest"
	"chain/core/txdb"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
)

// TestInsertVotingRightAccountID tests inserting a voting right into the
// database with a holder script that is the address of an account. The
// voting_rights row should contain the correct account id.
func TestInsertVotingRightAccountID(t *testing.T) {
	ctx := pgtest.NewContext(t)

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

	err := insertVotingRight(ctx, assetID, 0, 0, bc.Outpoint{}, data)
	if err != nil {
		t.Fatal(err)
	}

	// Look up the inserted voting right.
	var dbAccountID string
	err = pg.QueryRow(ctx, "SELECT account_id FROM voting_rights WHERE tx_hash = $1 AND index = $2", bc.Hash{}, 0).
		Scan(&dbAccountID)
	if err != nil {
		t.Fatal(err)
	}

	// The voting_rights row should have the correct account ID.
	if accountID != dbAccountID {
		t.Errorf("got=%s, want=%s", dbAccountID, accountID)
	}
}

// TestUpsertVotingToken tests inserting, updating and retrieving a voting
// token from the database index.
func TestUpsertVotingToken(t *testing.T) {
	ctx := pgtest.NewContext(t)

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

	err := upsertVotingToken(ctx, tokenAssetID, 0, out1, 100, data)
	if err != nil {
		t.Fatal(err)
	}

	// Modify the token state, and upsert it again.
	data.State, data.Vote = stateVoted, 2
	err = upsertVotingToken(ctx, tokenAssetID, 1, out2, 100, data)
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
			// All tokens distributed or registered.
			options: 5,
			seed: []testVoteToken{
				{state: stateDistributed, amount: 100, vote: 0},
				{state: stateRegistered, amount: 200, vote: 0},
				{state: stateRegistered, amount: 10, vote: 0},
				{state: stateDistributed, amount: 300, vote: 0},
				{state: stateDistributed, amount: 500, vote: 0},
				{state: stateRegistered, amount: 1, vote: 0},
			},
			want: Tally{
				Circulation: 1111,
				Distributed: 900,
				Registered:  211,
				Votes:       []int{0, 0, 0, 0, 0},
			},
		},
		{
			// Mix of voted, distributed and registered.
			options: 2,
			seed: []testVoteToken{
				{state: stateDistributed, amount: 100, vote: 0},
				{state: stateVoted, amount: 200, vote: 0},
				{state: stateRegistered, amount: 10, vote: 0},
				{state: stateDistributed, amount: 300, vote: 0},
				{state: stateVoted, amount: 500, vote: 1},
				{state: stateVoted, amount: 1, vote: 0},
			},
			want: Tally{
				Circulation: 1111,
				Distributed: 400,
				Registered:  10,
				Voted:       701,
				Votes:       []int{201, 500},
			},
		},
		{
			// Non-zero votes for tokens in states besides `stateVoted` should
			// not be tallied.
			options: 2,
			seed: []testVoteToken{
				{state: stateDistributed, amount: 499, vote: 1},
				{state: stateVoted, amount: 1, vote: 0},
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
				{state: stateVoted | stateFinished, amount: 100, vote: 0},
				{state: stateDistributed | stateFinished, amount: 10, vote: 0},
				{state: stateVoted | stateFinished, amount: 10000, vote: 1},
				{state: stateRegistered | stateFinished, amount: 3, vote: 0},
				{state: stateVoted | stateFinished, amount: 1000, vote: 0},
				{state: stateVoted | stateFinished, amount: 100, vote: 1},
			},
			want: Tally{
				Circulation: 11213,
				Distributed: 10,
				Registered:  3,
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
			err := upsertVotingToken(ctx, assetID, 1, bc.Outpoint{Index: uint32(i)}, vt.amount, tokenScriptData{
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

func TestGetVotesSimple(t *testing.T) {
	// TODO(jackson): Add additional tests for pagination, recalled voting
	// rights, voided voting rights, etc.
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB)) // TODO(kr): use memstore
	fc, err := assettest.InitializeSigningGenerator(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	Connect(fc)

	var (
		accountID1  = assettest.CreateAccountFixture(ctx, t, "", "", nil)
		accountID2  = assettest.CreateAccountFixture(ctx, t, "", "", nil)
		address1    = assettest.CreateAddressFixture(ctx, t, accountID1)
		address2    = assettest.CreateAddressFixture(ctx, t, accountID2)
		adminScript = assettest.CreateAddressFixture(ctx, t, accountID2).PKScript
		right1      = createVotingRightFixture(ctx, t, address1.PKScript)
		right2      = createVotingRightFixture(ctx, t, address2.PKScript)
		token1      = createVotingTokenFixture(ctx, t, right1.AssetID, adminScript, 100)
		token2      = createVotingTokenFixture(ctx, t, right2.AssetID, adminScript, 100)
		_           = createVotingTokenFixture(ctx, t, right1.AssetID, adminScript, 100)
	)

	tokens, last, err := GetVotes(ctx, []bc.AssetID{token1.AssetID, token2.AssetID}, accountID1, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 {
		t.Errorf("got %v tokens, want 1", len(tokens))
	}
	wantLast := fmt.Sprintf("%s-%s", token1.AssetID, right1.AssetID)
	if last != wantLast {
		t.Errorf("last: got=%s, want=%s", last, wantLast)
	}
	token1.AccountID = accountID1
	if !reflect.DeepEqual(tokens[0], token1) {
		t.Errorf("tokens[0]: got=%#v, want=%#v", tokens[0], token1)
	}
}
