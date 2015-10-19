package asset

import (
	"testing"

	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestTransfer(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO manager_nodes (id, project_id, label, current_rotation)
			VALUES('mn1', 'proj-id-0', 'mn1', 'rot1');
		INSERT INTO rotations (id, manager_node_id, keyset)
			VALUES('rot1', 'mn1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}');
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('b1', 'mn1', 0, 1);
		INSERT INTO addresses (id, manager_node_id, account_id, keyset, key_index, address, redeem_script, pk_script)
			VALUES('a1', 'mn1', 'b1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0, 'a1', '', '');
		INSERT INTO utxos (txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
			VALUES ('246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7', 0, 'ff00000000000000000000000000000000000000000000000000000000000000', 6, 0, 'b1', 'mn1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	_, err := Transfer(ctx,
		[]utxodb.Input{{
			BucketID: "b1",
			AssetID:  "ff00000000000000000000000000000000000000000000000000000000000000",
			Amount:   5,
		}},
		[]*Output{{
			AssetID: "ff00000000000000000000000000000000000000000000000000000000000000",
			Address: "3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB",
			Amount:  5,
		}},
	)

	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
}

func TestValidateOutputs(t *testing.T) {
	cases := []struct {
		outs    []*Output
		wantErr error
	}{{
		outs:    []*Output{{AssetID: "x", Amount: 5, BucketID: "b1", Address: "a"}},
		wantErr: ErrBadOutDest,
	}, {
		outs:    []*Output{{AssetID: "x", Amount: 5}},
		wantErr: ErrBadOutDest,
	}, {
		outs:    []*Output{{AssetID: "x", Amount: 5, BucketID: "b1"}},
		wantErr: nil,
	}}

	for _, c := range cases {
		got := validateOutputs(c.outs)

		if errors.Root(got) != c.wantErr {
			t.Errorf("got err = %v want %v", errors.Root(got), c.wantErr)
		}
	}
}

func TestCheckTransferParity(t *testing.T) {
	cases := []struct {
		ins  []utxodb.Input
		outs []*Output
		want error
	}{{
		ins:  []utxodb.Input{{AssetID: "x", Amount: 4}},
		outs: []*Output{},
		want: ErrBadTx,
	}, {
		ins:  []utxodb.Input{},
		outs: []*Output{{AssetID: "x", Amount: 4}},
		want: ErrBadTx,
	}, {
		ins:  []utxodb.Input{{AssetID: "x", Amount: 4}},
		outs: []*Output{{AssetID: "y", Amount: 4}},
		want: ErrBadTx,
	}, {
		ins:  []utxodb.Input{{AssetID: "x", Amount: 4}},
		outs: []*Output{{AssetID: "x", Amount: 5}},
		want: ErrBadTx,
	}, {
		ins:  []utxodb.Input{{AssetID: "x", Amount: 4}},
		outs: []*Output{{AssetID: "x", Amount: 4}},
		want: nil,
	}}

	for _, c := range cases {
		err := checkTransferParity(c.ins, c.outs)
		if errors.Root(err) != c.want {
			t.Errorf("got err = %q want %q", errors.Root(err), c.want)
		}
	}
}
