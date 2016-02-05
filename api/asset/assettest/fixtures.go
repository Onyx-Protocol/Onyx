package assettest

import (
	"fmt"
	"testing"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/api/txdb"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/testutil"
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
		testutil.FatalErr(t, err)
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
		testutil.FatalErr(t, err)
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
		xpubs = append(xpubs, testutil.TestXPub)
		xprvs = append(xprvs, testutil.TestXPrv)
	}
	issuerNode, err := appdb.InsertIssuerNode(ctx, projectID, label, xpubs, xprvs, 1)
	if err != nil {
		testutil.FatalErr(t, err)
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
		xpubs = append(xpubs, testutil.TestXPub)
		xprvs = append(xprvs, testutil.TestXPrv)
	}
	managerNode, err := appdb.InsertManagerNode(ctx, projectID, label, xpubs, xprvs, 0, 1)
	if err != nil {
		testutil.FatalErr(t, err)
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
		testutil.FatalErr(t, err)
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
		testutil.FatalErr(t, err)
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

func NewContextWithGenesisBlock(tb testing.TB) context.Context {
	ctx := pgtest.NewContext(tb)

	key, err := testutil.TestXPrv.ECPrivKey()
	if err != nil {
		tb.Fatal(err)
	}
	asset.BlockKey = key

	_, err = asset.UpsertGenesisBlock(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	return ctx
}

var opIndexCounter = createCounter()

func CreateAccountUTXOFixture(ctx context.Context, t *testing.T, accountID string, assetID bc.AssetID, amt uint64, confirmed bool) bc.Outpoint {
	if accountID == "" {
		accountID = CreateAccountFixture(ctx, t, "", "x", []string{testutil.TestXPub.String()})
	}
	addr := &appdb.Address{AccountID: accountID}
	err := appdb.CreateAddress(ctx, addr, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	output := &txdb.Output{
		Output: state.Output{
			Outpoint: bc.Outpoint{Index: uint32(<-opIndexCounter)},
			TxOutput: bc.TxOutput{
				AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: amt},
			},
		},
		ManagerNodeID: addr.ManagerNodeID,
		AccountID:     accountID,
	}
	copy(output.AddrIndex[:], addr.Index[0:2])

	if !confirmed {
		// ignore error from potential duplicate
		txdb.InsertPoolTx(ctx, &bc.Tx{Hash: output.Outpoint.Hash, TxData: bc.TxData{}})
		err = txdb.InsertPoolOutputs(ctx, []*txdb.Output{output})
		if err != nil {
			testutil.FatalErr(t, err)
		}
	} else {
		_, err = txdb.InsertBlockOutputs(ctx, &bc.Block{}, []*txdb.Output{output})
	}
	return output.Outpoint
}
