package voting

import (
	"os"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset/assettest"
	"chain/cos/bc"
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
