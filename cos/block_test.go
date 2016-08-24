package cos

import (
	"context"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"chain/cos/bc"
	"chain/cos/mempool"
	"chain/cos/memstore"
	"chain/cos/state"
	"chain/cos/vm"
	"chain/crypto/ed25519"
	"chain/errors"
	"chain/testutil"
)

func TestLatestBlock(t *testing.T) {
	ctx := context.Background()

	emptyPool := mempool.New()
	noBlocks := memstore.New()
	oneBlock := memstore.New()
	oneBlock.SaveBlock(ctx, &bc.Block{})
	oneBlock.SaveSnapshot(ctx, 1, state.Empty())

	cases := []struct {
		store   Store
		want    *bc.Block
		wantErr error
	}{
		{noBlocks, nil, ErrNoBlocks},
		{oneBlock, &bc.Block{}, nil},
	}

	for _, c := range cases {
		fc, err := NewFC(ctx, c.store, emptyPool, nil, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		got, gotErr := fc.LatestBlock(ctx)

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("got latest = %+v want %+v", got, c.want)
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("got latest err = %q want %q", gotErr, c.wantErr)
		}
	}
}

func TestNoTimeTravel(t *testing.T) {
	ctx := context.Background()
	fc, err := NewFC(ctx, memstore.New(), mempool.New(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	fc.setHeight(1)
	fc.setHeight(2)

	fc.setHeight(1) // don't go backward
	if fc.height.n != 2 {
		t.Fatalf("fc.height.n = %d want 2", fc.height.n)
	}
}

func TestWaitForBlock(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()
	block1 := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Height:           1,
			ConsensusProgram: []byte{byte(vm.OP_TRUE)},
		},
	}
	block2 := &bc.Block{
		BlockHeader: bc.BlockHeader{
			PreviousBlockHash: block1.Hash(),
			Height:            2,
			ConsensusProgram:  []byte{byte(vm.OP_TRUE)},
		},
	}
	block3 := &bc.Block{
		BlockHeader: bc.BlockHeader{
			PreviousBlockHash: block2.Hash(),
			Height:            3,
			ConsensusProgram:  []byte{byte(vm.OP_TRUE)},
		},
	}
	store.SaveBlock(ctx, block1)
	store.SaveSnapshot(ctx, 1, state.Empty())
	fc, err := NewFC(ctx, store, mempool.New(), nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	ch := waitForBlockChan(ctx, fc, 1)
	select {
	case err := <-ch:
		if err != nil {
			t.Errorf("got err %q waiting for block 0", err)
		}
	case <-time.After(10 * time.Millisecond):
		t.Errorf("timed out waiting for block 0")
	}

	ch = waitForBlockChan(ctx, fc, 5)
	select {
	case err := <-ch:
		if err != ErrTheDistantFuture {
			t.Errorf("got %q waiting for block 5, expected %q", err, ErrTheDistantFuture)
		}
	case <-time.After(10 * time.Millisecond):
		t.Errorf("timed out waiting for block 5")
	}

	ch = waitForBlockChan(ctx, fc, 2)

	select {
	case <-ch:
		t.Errorf("WaitForBlock should wait")
	default:
	}

	err = fc.CommitBlock(ctx, block2, state.Empty())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	select {
	case <-ch:
		t.Errorf("WaitForBlock should wait")
	default:
	}

	err = fc.CommitBlock(ctx, block3, state.Empty())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	select {
	case err := <-ch:
		if err != nil {
			t.Errorf("got err %q waiting for block 3", err)
		}
	case <-time.After(10 * time.Millisecond):
		t.Errorf("timed out waiting for block 3")
	}
}

func waitForBlockChan(ctx context.Context, fc *FC, height uint64) chan error {
	ch := make(chan error)
	go func() {
		err := fc.WaitForBlock(ctx, height)
		ch <- err
	}()
	return ch
}

func TestIdempotentUpsert(t *testing.T) {
	ctx, fc := newContextFC(t)

	// InitializeSigningGenerator added a genesis block.  Calling
	// UpsertGenesisBlock again should be a no-op, not produce an error.
	for i := 0; i < 2; i++ {
		var err error
		_, err = fc.UpsertGenesisBlock(ctx, testutil.TestPubs, 1, time.Now())
		if err != nil {
			testutil.FatalErr(t, err)
		}
	}
}

func TestGenerateBlock(t *testing.T) {
	ctx, fc := newContextFC(t)

	now := time.Unix(233400000, 0)

	latestBlock, err := fc.UpsertGenesisBlock(ctx, testutil.TestPubs, 1, now)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	genesisHash := latestBlock.Hash()
	assetID := bc.ComputeAssetID(nil, genesisHash, 1)

	txs := []*bc.Tx{
		bc.NewTx(bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{
				bc.NewIssuanceInput(now, now.Add(time.Hour), genesisHash, 50, nil, nil, [][]byte{
					nil,
					mustDecodeHex("30450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b5101"),
					mustDecodeHex("51210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae")}),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(assetID, 50, mustDecodeHex("a9145881cd104f8d64635751ac0f3c0decf9150c110687"), nil),
			},
		}),
		bc.NewTx(bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{
				bc.NewIssuanceInput(now, now.Add(time.Hour), genesisHash, 50, nil, nil, [][]byte{
					nil,
					mustDecodeHex("3045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e01"),
					mustDecodeHex("51210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae"),
				}),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(assetID, 50, mustDecodeHex("a914c171e443e05b953baa7b7d834028ed91e47b4d0b87"), nil),
			},
		}),
	}
	for _, tx := range txs {
		err := fc.pool.Insert(ctx, tx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
	}

	got, err := fc.GenerateBlock(ctx, latestBlock, state.Empty(), now)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}

	// TODO(bobg): verify these hashes are correct
	var wantTxRoot, wantAssetsRoot bc.Hash
	copy(wantTxRoot[:], mustDecodeHex("60154ef1bb64c1f7567184ca6ae3cd2645788ffe3c723f9a8846c4a3a87ba9bf"))
	copy(wantAssetsRoot[:], mustDecodeHex("a88ef8959187a0c5905beed2019269835bca245fbd36b09b95f684b5038bdf93"))

	want := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:                bc.NewBlockVersion,
			Height:                 2,
			PreviousBlockHash:      latestBlock.Hash(),
			TransactionsMerkleRoot: wantTxRoot,
			AssetsMerkleRoot:       wantAssetsRoot,
			TimestampMS:            bc.Millis(now),
			ConsensusProgram:       latestBlock.ConsensusProgram,
		},
		Transactions: txs,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
	}

}

func TestValidateGenesisBlockForSig(t *testing.T) {
	genesis, err := NewGenesisBlock(testutil.TestPubs, 1, time.Now())
	if err != nil {
		t.Fatal("unexpected error ", err)
	}

	ctx := context.Background()
	fc, err := NewFC(ctx, memstore.New(), mempool.New(), nil, nil)
	if err != nil {
		t.Fatal("unexpected error ", err)
	}

	err = fc.ValidateBlockForSig(ctx, genesis)
	if err != nil {
		t.Error("unexpected error ", err)
	}
}

func TestIsSignedByTrustedHost(t *testing.T) {
	privKey := testutil.TestPrv
	privKeys := []ed25519.PrivateKey{privKey}

	block := &bc.Block{}
	signBlock(t, block, privKeys)

	cases := []struct {
		desc        string
		witness     [][]byte
		trustedKeys []ed25519.PublicKey
		want        bool
	}{{
		desc:        "empty sig",
		witness:     nil,
		trustedKeys: privToPub(privKeys),
		want:        false,
	}, {
		desc:        "wrong trusted keys",
		witness:     block.Witness,
		trustedKeys: privToPub([]ed25519.PrivateKey{newPrivKey(t)}),
		want:        false,
	}, {
		desc:        "one-of-one trusted keys",
		witness:     block.Witness,
		trustedKeys: privToPub(privKeys),
		want:        true,
	}, {
		desc:        "one-of-two trusted keys",
		witness:     block.Witness,
		trustedKeys: privToPub(append(privKeys, newPrivKey(t))),
		want:        true,
	}}

	for _, c := range cases {
		block.Witness = c.witness
		got := isSignedByTrustedHost(block, c.trustedKeys)

		if got != c.want {
			t.Errorf("%s: got %v want %v", c.desc, got, c.want)
		}
	}
}

func newContextFC(t testing.TB) (context.Context, *FC) {
	ctx := context.Background()
	fc, err := NewFC(ctx, memstore.New(), mempool.New(), nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return ctx, fc
}

func signBlock(t testing.TB, b *bc.Block, keys []ed25519.PrivateKey) {
	var sigs [][]byte
	for _, key := range keys {
		sig := ComputeBlockSignature(b, key)
		sigs = append(sigs, sig)
	}
	AddSignaturesToBlock(b, sigs)
}

func privToPub(privs []ed25519.PrivateKey) []ed25519.PublicKey {
	var public []ed25519.PublicKey
	for _, priv := range privs {
		public = append(public, priv.Public().(ed25519.PublicKey))
	}
	return public
}

func newPrivKey(t *testing.T) ed25519.PrivateKey {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return priv
}

func mustDecodeHex(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}
