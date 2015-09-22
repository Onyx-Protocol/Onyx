package asset

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestTransfer(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES ('app-id-0', 'app-0');
		INSERT INTO wallets (id, application_id, label, current_rotation)
			VALUES('w1', 'app-id-0', 'w1', 'rot1');
		INSERT INTO rotations (id, wallet_id, keyset)
			VALUES('rot1', 'w1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}');
		INSERT INTO buckets (id, wallet_id, key_index, next_address_index)
			VALUES('b1', 'w1', 0, 1);
		INSERT INTO addresses (id, wallet_id, bucket_id, keyset, key_index, address, redeem_script, pk_script)
			VALUES('a1', 'w1', 'b1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0, 'a1', '', '');
		INSERT INTO utxos (txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
			VALUES ('246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7', 0, 'AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx', 6, 'a1', 'b1', 'w1');
	`)
	defer dbtx.Rollback()

	ctx := pg.NewContext(context.Background(), dbtx)
	_, err := Transfer(ctx,
		[]TransferInput{{
			BucketID: "b1",
			AssetID:  "AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx",
			Amount:   5,
		}},
		[]Output{{
			AssetID: "AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx",
			Address: "3H9gBofbYu4uQXwfMVcFiWjQHXf6vmnVGB",
			Amount:  5,
		}},
	)

	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
}

func TestValidateTransfer(t *testing.T) {
	cases := []struct {
		ins     []TransferInput
		outs    []Output
		wantErr error
	}{{
		ins:     []TransferInput{},
		outs:    []Output{{AssetID: "x", Amount: 5, BucketID: "b1"}},
		wantErr: ErrTransferMismatch,
	}, {
		ins:     []TransferInput{{AssetID: "x", Amount: 5}},
		outs:    []Output{{AssetID: "x", Amount: 5, BucketID: "b1", Address: "a"}},
		wantErr: ErrBadOutDest,
	}, {
		ins:     []TransferInput{{AssetID: "x", Amount: 5}},
		outs:    []Output{{AssetID: "x", Amount: 5}},
		wantErr: ErrBadOutDest,
	}, {
		ins:     []TransferInput{{AssetID: "x", Amount: 5}},
		outs:    []Output{{AssetID: "x", Amount: 5, BucketID: "b1"}},
		wantErr: nil,
	}}

	for _, c := range cases {
		got := validateTransfer(c.ins, c.outs)

		if errors.Root(got) != c.wantErr {
			t.Errorf("got err = %v want %v", errors.Root(got), c.wantErr)
		}
	}
}
