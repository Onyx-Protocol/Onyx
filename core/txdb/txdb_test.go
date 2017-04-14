package txdb

import (
	"bytes"
	"context"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/state"
	"chain/testutil"
)

func TestLatestSnapshot(t *testing.T) {
	ctx := context.Background()
	dbtx := pgtest.NewTx(t)
	store := NewStore(dbtx)

	snap := state.Empty()
	snap.Nonces[bc.NewHash([32]byte{0xc0, 0x01})] = 12345678
	err := snap.Tree.Insert([]byte{0x01, 0x02, 0x03, 0x04})
	if err != nil {
		t.Fatal(err)
	}
	err = store.SaveSnapshot(ctx, 5, snap)
	if err != nil {
		t.Fatal(err)
	}

	// Check that LatestSnapshotInfo returns the info for the new snapshot.
	height, size, err := store.LatestSnapshotInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if height != 5 {
		t.Errorf("LatestSnapshotInfo height got %d, want 5", height)
	}
	// Check that LatestSnapshot returns the same snapshot.
	got, height, err := store.LatestSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if height != 5 {
		t.Errorf("LatestSnapshotInfo height got %d, want 5", height)
	}
	if !testutil.DeepEqual(got, snap) {
		t.Errorf("LatestSnapshot got %#v want %#v", got, snap)
	}
	// Check that GetSnapshot returns the raw bytes of the same snapshot.
	raw, err := store.GetSnapshot(ctx, height)
	if err != nil {
		t.Fatal(err)
	}
	if uint64(len(raw)) != size {
		t.Errorf("GetSnapshot returned %d-byte snapshot, info said it was %d bytes", size, len(raw))
	}
	decoded, err := DecodeSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !testutil.DeepEqual(decoded, snap) {
		t.Errorf("GetSnapshot got %#v, want %#v", decoded, snap)
	}
}

func TestGetRawBlock(t *testing.T) {
	ctx := context.Background()
	dbtx := pgtest.NewTx(t)

	block := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            10,
			PreviousBlockHash: bc.NewHash([32]byte{0x09}),
			TimestampMS:       123456,
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: bc.NewHash([32]byte{0x01}),
				AssetsMerkleRoot:       bc.NewHash([32]byte{0x02}),
				ConsensusProgram:       []byte{0xc0, 0x01},
			},
			BlockWitness: legacy.BlockWitness{
				Witness: [][]byte{[]byte{0xbe, 0xef}},
			},
		},
	}
	var buf bytes.Buffer
	_, err := block.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}

	store := NewStore(dbtx)
	err = store.SaveBlock(ctx, block)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := store.GetRawBlock(ctx, block.Height)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), raw) {
		t.Errorf("GetRawBlock got %x, want %x", raw, buf.Bytes())
	}
}

func TestListenFinalizeBlocks(t *testing.T) {
	dbURL, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := NewStore(db)

	// Start listening for new blocks.
	heightCh, err := ListenBlocks(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}

	err = store.FinalizeBlock(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	height := <-heightCh
	if height != 1 {
		t.Errorf("heightCh: got %d want 1", height)
	}
}

func TestGetBlock(t *testing.T) {
	ctx := context.Background()
	dbtx := pgtest.NewTx(t)
	pgtest.Exec(ctx, dbtx, t, `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES
		(decode('0000000000000000000000000000000000000000000000000000000000000000', 'hex'), 0, '', ''),
		(
			decode('1f20d89dd393f452b4396589ed5d6f90465cb032aa3f9fe42a99d47c7089b0a3', 'hex'),
			1,
			decode('03010131323300000000000000000000000000000000000000000000000000000000006453414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000012746573742d6f75747075742d73637269707411010f746573742d7369672d73637269707401070102000000000007746573742d7478', 'hex'),
			''
		);
	`)
	store := NewStore(dbtx)
	got, err := store.GetBlock(ctx, 1)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}
	want := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            1,
			PreviousBlockHash: bc.NewHash([32]byte{'1', '2', '3'}),
			TimestampMS:       100,
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: bc.NewHash([32]byte{'A', 'B', 'C'}),
				AssetsMerkleRoot:       bc.NewHash([32]byte{'X', 'Y', 'Z'}),
				ConsensusProgram:       []byte("test-output-script"),
			},
			BlockWitness: legacy.BlockWitness{
				Witness: [][]byte{[]byte("test-sig-script")},
			},
		},
		Transactions: []*legacy.Tx{
			legacy.NewTx(legacy.TxData{Version: 1, ReferenceData: []byte("test-tx")}),
		},
	}

	if !testutil.DeepEqual(got, want) {
		t.Errorf("latest block:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestInsertBlock(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	blk := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version: 1,
			Height:  1,
		},
		Transactions: []*legacy.Tx{
			legacy.NewTx(legacy.TxData{
				ReferenceData: []byte("a"),
			}),
			legacy.NewTx(legacy.TxData{
				ReferenceData: []byte("b"),
			}),
		},
	}
	s := NewStore(dbtx)
	err := s.SaveBlock(ctx, blk)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	got, err := s.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if !testutil.DeepEqual(got, blk) {
		t.Errorf("got %#v, wanted %#v", got, blk)
	}
}
