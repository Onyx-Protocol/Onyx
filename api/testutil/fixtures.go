package testutil

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
)

var userCounter = createCounter()

func CreateUserFixture(ctx context.Context, t *testing.T, email, password string) string {
	if email == "" {
		email = fmt.Sprintf("user-%d@domain.tld", <-userCounter)
	}
	if password == "" {
		password = "drowssap"
	}
	user, err := appdb.CreateUser(ctx, email, password)
	if err != nil {
		FatalErr(t, err)
	}
	return user.ID
}

var projCounter = createCounter()

func CreateProjectFixture(ctx context.Context, t *testing.T, userID, name string) string {
	if userID == "" {
		userID = CreateUserFixture(ctx, t, "", "")
	}
	if name == "" {
		name = fmt.Sprintf("proj-%d", <-projCounter)
	}
	proj, err := appdb.CreateProject(ctx, name, userID)
	if err != nil {
		FatalErr(t, err)
	}
	return proj.ID
}

var issuerNodeCounter = createCounter()

func CreateIssuerNodeFixture(ctx context.Context, t *testing.T, projectID, label string, xpubs, xprvs []*hdkey.XKey) string {
	if projectID == "" {
		projectID = CreateProjectFixture(ctx, t, "", "")
	}
	if label == "" {
		label = fmt.Sprintf("inode-%d", <-issuerNodeCounter)
	}
	if len(xpubs) == 0 && len(xprvs) == 0 {
		xpubs = append(xpubs, TestXPub)
		xprvs = append(xprvs, TestXPrv)
	}
	issuerNode, err := appdb.InsertIssuerNode(ctx, projectID, label, xpubs, xprvs, 1)
	if err != nil {
		FatalErr(t, err)
	}
	return issuerNode.ID
}

var managerNodeCounter = createCounter()

func CreateManagerNodeFixture(ctx context.Context, t *testing.T, projectID, label string, xpubs, xprvs []*hdkey.XKey) string {
	if projectID == "" {
		projectID = CreateProjectFixture(ctx, t, "", "")
	}
	if label == "" {
		label = fmt.Sprintf("mnode-%d", <-managerNodeCounter)
	}
	if len(xpubs) == 0 && len(xprvs) == 0 {
		xpubs = append(xpubs, TestXPub)
		xprvs = append(xprvs, TestXPrv)
	}
	managerNode, err := appdb.InsertManagerNode(ctx, projectID, label, xpubs, xprvs, 0, 1)
	if err != nil {
		FatalErr(t, err)
	}
	return managerNode.ID
}

var accountCounter = createCounter()

func CreateAccountFixture(ctx context.Context, t *testing.T, managerNodeID, label string, keys []string) string {
	if managerNodeID == "" {
		managerNodeID = CreateManagerNodeFixture(ctx, t, "", "", nil, nil)
	}
	if label == "" {
		label = fmt.Sprintf("acct-%d", <-accountCounter)
	}
	account, err := appdb.CreateAccount(ctx, managerNodeID, label, keys)
	if err != nil {
		FatalErr(t, err)
	}
	return account.ID
}

var assetCounter = createCounter()

func CreateAssetFixture(ctx context.Context, t *testing.T, issuerNodeID, label string) bc.AssetID {
	if issuerNodeID == "" {
		issuerNodeID = CreateIssuerNodeFixture(ctx, t, "", "", nil, nil)
	}
	if label == "" {
		label = fmt.Sprintf("inode-%d", <-assetCounter)
	}
	asset, err := asset.Create(ctx, issuerNodeID, label, nil)
	if err != nil {
		FatalErr(t, err)
	}
	return asset.Hash
}

// Creates an infinite stream of integers counting up from 1
func createCounter() <-chan int {
	result := make(chan int)
	go func() {
		var n int
		for true {
			n++
			result <- n
		}
	}()
	return result
}

func CreateGenesisBlock(ctx context.Context) {
	genesisBlock := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:   bc.NewBlockVersion,
			Timestamp: uint64(time.Now().Unix()),
		},
	}
	const q = `
		INSERT INTO blocks (block_hash, height, data, header)
		    VALUES ($1, $2, $3, $4)
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, genesisBlock.Hash(), genesisBlock.Height, genesisBlock, &genesisBlock.BlockHeader)
	if err != nil {
		panic(err)
	}
}

func NewContextWithGenesisBlock(tb testing.TB, sql ...string) context.Context {
	ctx := pgtest.NewContext(tb, sql...)
	CreateGenesisBlock(ctx)
	return ctx
}
