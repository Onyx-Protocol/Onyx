package voting

import (
	"os"
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain/bc"
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
		projectID     = assettest.CreateProjectFixture(ctx, t, "", "")
		managerNodeID = assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
		issuerNodeID  = assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
		accountID     = assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)
		assetID       = assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
		address       = assettest.CreateAddressFixture(ctx, t, accountID)
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
