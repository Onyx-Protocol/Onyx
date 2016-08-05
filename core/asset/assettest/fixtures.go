package assettest

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/core/appdb"
	"chain/core/asset"
	"chain/core/blocksigner"
	"chain/core/generator"
	"chain/core/txbuilder"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/mempool"
	"chain/cos/memstore"
	"chain/cos/state"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/errors"
	"chain/testutil"
)

var userCounter = createCounter()

func CreateUserFixture(ctx context.Context, t testing.TB, email, password, role string) string {
	if email == "" {
		email = fmt.Sprintf("user-%d@domain.tld", <-userCounter)
	}
	if password == "" {
		password = "drowssap"
	}
	if role == "" {
		role = "developer"
	}
	user, err := appdb.CreateUser(ctx, email, password, role)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return user.ID
}

func CreateAuthTokenFixture(ctx context.Context, t testing.TB, userID string, typ string, expiresAt *time.Time) *appdb.AuthToken {
	token, err := appdb.CreateAuthToken(ctx, userID, typ, expiresAt)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return token
}

func CreateInvitationFixture(ctx context.Context, t testing.TB, email, role string) string {
	invitation, err := appdb.CreateInvitation(ctx, email, role)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return invitation.ID
}

func CreateAccountFixture(ctx context.Context, t testing.TB, keys []string, quorum int, tags map[string]interface{}) string {
	if keys == nil {
		keys = []string{testutil.TestXPub.String()}
	}
	if quorum == 0 {
		quorum = len(keys)
	}
	acc, err := account.Create(ctx, keys, quorum, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return acc.ID
}

func CreateAccountControlProgramFixture(ctx context.Context, t testing.TB, accID string) []byte {
	if accID == "" {
		accID = CreateAccountFixture(ctx, t, nil, 0, nil)
	}
	controlProgram, err := account.CreateControlProgram(ctx, accID)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return controlProgram
}

var assetCounter = createCounter()

func CreateAssetFixture(ctx context.Context, t testing.TB, keys []string, quorum int, def map[string]interface{}) bc.AssetID {
	if len(keys) == 0 {
		keys = []string{testutil.TestXPub.String()}
	}

	if quorum == 0 {
		quorum = len(keys)
	}
	var genesisHash bc.Hash

	asset, err := asset.Define(ctx, keys, quorum, def, genesisHash, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return asset.AssetID
}

// Creates an infinite stream of integers counting up from 1
func createCounter() <-chan int {
	result := make(chan int)
	go func() {
		var n int
		for {
			n++
			result <- n
		}
	}()
	return result
}

func IssueAssetsFixture(ctx context.Context, t testing.TB, assetID bc.AssetID, amount uint64, accountID string) state.Output {
	if accountID == "" {
		accountID = CreateAccountFixture(ctx, t, nil, 0, nil)
	}
	dest := AccountDestinationFixture(ctx, t, assetID, amount, accountID)

	assetAmount := bc.AssetAmount{AssetID: assetID, Amount: amount}

	src := asset.NewIssueSource(ctx, assetAmount, nil) // does not support reference data
	tpl, err := txbuilder.Build(ctx, nil, []*txbuilder.Source{src}, []*txbuilder.Destination{dest}, nil, time.Minute)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	SignTxTemplate(t, tpl, testutil.TestXPrv)

	tx, err := asset.FinalizeTx(ctx, tpl)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return state.Output{
		Outpoint: bc.Outpoint{Hash: tx.Hash, Index: 0},
		TxOutput: *tx.Outputs[0],
	}
}

func AccountDestinationFixture(ctx context.Context, t testing.TB, assetID bc.AssetID, amount uint64, accountID string) *txbuilder.Destination {
	dest, err := account.NewDestination(ctx, &bc.AssetAmount{AssetID: assetID, Amount: amount}, accountID, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return dest
}

// InitializeSigningGenerator initiaizes a generator fixture with the
// provided store. Store can be nil, in which case it will use memstore.
func InitializeSigningGenerator(ctx context.Context, store cos.Store, pool cos.Pool) (*cos.FC, *generator.Generator, error) {
	if store == nil {
		store = memstore.New()
	}
	if pool == nil {
		pool = mempool.New()
	}
	fc, err := cos.NewFC(ctx, store, pool, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	asset.Init(fc, true)
	account.Init(fc)
	privkey := testutil.TestPrv
	localSigner := blocksigner.New(privkey, pg.FromContext(ctx), fc)
	g := &generator.Generator{
		Config: generator.Config{
			LocalSigner:  localSigner,
			BlockPeriod:  time.Second,
			BlockKeys:    []ed25519.PublicKey{testutil.TestPub},
			SigsRequired: 1,
			FC:           fc,
		},
	}
	if err != nil {
		return nil, nil, err
	}
	err = g.UpsertGenesisBlock(ctx)
	if err != nil {
		return nil, nil, err
	}
	return fc, g, nil
}

func Issue(ctx context.Context, t testing.TB, assetID bc.AssetID, dests []*txbuilder.Destination) *bc.Tx {
	var issueAmount uint64
	for _, dst := range dests {
		if dst.AssetID != assetID {
			continue
		}
		issueAmount += dst.Amount
	}

	assetAmount := bc.AssetAmount{AssetID: assetID, Amount: issueAmount}

	txTemplate, err := txbuilder.Build(
		ctx,
		nil,
		[]*txbuilder.Source{asset.NewIssueSource(ctx, assetAmount, nil)},
		dests,
		nil,
		time.Minute,
	)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	SignTxTemplate(t, txTemplate, nil)
	tx, err := asset.FinalizeTx(ctx, txTemplate)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	return tx
}

func Transfer(ctx context.Context, t testing.TB, srcs []*txbuilder.Source, dests []*txbuilder.Destination) *bc.Tx {
	template, err := txbuilder.Build(ctx, nil, srcs, dests, nil, time.Hour)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	SignTxTemplate(t, template, testutil.TestXPrv)

	tx, err := asset.FinalizeTx(ctx, template)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	return tx
}

func AccountDest(ctx context.Context, t testing.TB, accountID string, assetID bc.AssetID, amount uint64) *txbuilder.Destination {
	d, err := account.NewDestination(ctx, &bc.AssetAmount{
		AssetID: assetID,
		Amount:  amount,
	}, accountID, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	return d
}
