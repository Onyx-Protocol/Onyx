package asset

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/wire"
)

func TestTransfer(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES ('app-id-0', 'app-0');
		INSERT INTO keys (id, xpub) VALUES(
			'fda6bac8e1901cbc4813e729d3d766988b8b1ac7',
			'xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd'
		);
		INSERT INTO wallets (id, application_id, label, current_rotation)
			VALUES('w1', 'app-id-0', 'w1', 'rot1');
		INSERT INTO rotations (id, wallet_id, keyset)
			VALUES('rot1', 'w1', '{fda6bac8e1901cbc4813e729d3d766988b8b1ac7}');
		INSERT INTO buckets (id, wallet_id, key_index, next_address_index)
			VALUES('b1', 'w1', 0, 1);
		INSERT INTO addresses (id, wallet_id, bucket_id, keyset, key_index, address, redeem_script, pk_script)
			VALUES('a1', 'w1', 'b1', '{fda6bac8e1901cbc4813e729d3d766988b8b1ac7}', 0, 'a1', '', '');
		INSERT INTO utxos (txid, index, asset_id, amount, address_id, bucket_id, wallet_id)
			VALUES ('246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7', 0, 'AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx', 6, 'a1', 'b1', 'w1');
	`)
	defer dbtx.Rollback()

	ctx := pg.NewContext(context.Background(), dbtx)
	tx, err := Transfer(ctx,
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

	wantTx := wire.NewMsgTx()

	inHash, _ := wire.NewHash32FromStr("246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7")
	wantTx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(inHash, 0), []byte{}))

	outAsset, _ := wire.NewHash20FromStr("AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx")
	outScript, _ := hex.DecodeString("a914a994a46855d8f4442b3a6db863628cc020537f4087")
	wantTx.AddTxOut(wire.NewTxOut(outAsset, 5, outScript))

	outScript, _ = hex.DecodeString("a914613333cd9c090b6ea455fb0a894f1824f4dc74f187")
	wantTx.AddTxOut(wire.NewTxOut(outAsset, 1, outScript))

	gotTx := wire.NewMsgTx()
	err = gotTx.Deserialize(bytes.NewBuffer(tx.Unsigned))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotTx, wantTx) {
		t.Errorf("got tx = %+v want %+v", gotTx, wantTx)
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
