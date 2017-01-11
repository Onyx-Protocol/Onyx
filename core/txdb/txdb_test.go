package txdb

import (
	"context"
	"reflect"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/testutil"
)

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
	want := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           1,
			Height:            1,
			PreviousBlockHash: [32]byte{'1', '2', '3'},
			TransactionsMerkleRoot: bc.Hash{
				'A', 'B', 'C', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
			AssetsMerkleRoot: bc.Hash{
				'X', 'Y', 'Z', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
			TimestampMS:      100,
			Witness:          [][]byte{[]byte("test-sig-script")},
			ConsensusProgram: []byte("test-output-script"),
		},
		Transactions: []*bc.Tx{
			bc.NewTx(bc.TxData{Version: 1, ReferenceData: []byte("test-tx")}),
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("latest block:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestInsertBlock(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	blk := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version: 1,
			Height:  1,
		},
		Transactions: []*bc.Tx{
			bc.NewTx(bc.TxData{
				ReferenceData: []byte("a"),
			}),
			bc.NewTx(bc.TxData{
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
	if !reflect.DeepEqual(got, blk) {
		t.Errorf("got %#v, wanted %#v", got, blk)
	}
}
