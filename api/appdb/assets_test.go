package appdb

import (
	"encoding/hex"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/wire"
)

func TestAssetByID(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleAppFixture, `
		INSERT INTO keys (id, xpub)
		VALUES(
			'fda6bac8e1901cbc4813e729d3d766988b8b1ac7',
			'xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd'
		);
		INSERT INTO asset_groups (id, application_id, label, keyset, key_index)
			VALUES ('ag1', 'app-id-0', 'foo', '{fda6bac8e1901cbc4813e729d3d766988b8b1ac7}', 0);
		INSERT INTO assets (id, asset_group_id, key_index, keyset, redeem_script, label)
		VALUES(
			'AU8RjUUysqep9wXcZKqtTty1BssV6TcX7p',
			'ag1',
			0,
			'{fda6bac8e1901cbc4813e729d3d766988b8b1ac7}',
			decode('51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae', 'hex'),
			'foo'
		);
	`)
	defer dbtx.Rollback()

	ctx := pg.NewContext(context.Background(), dbtx)
	got, err := AssetByID(ctx, "AU8RjUUysqep9wXcZKqtTty1BssV6TcX7p")
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	hash, _ := wire.NewHash20FromStr("AU8RjUUysqep9wXcZKqtTty1BssV6TcX7p")
	redeem, _ := hex.DecodeString("51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae")
	key, _ := NewKey("xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd")
	want := &Asset{
		Hash:         hash,
		GroupID:      "ag1",
		AGIndex:      []uint32{0, 0},
		AIndex:       []uint32{0, 0},
		RedeemScript: redeem,
		Keys:         []*Key{key},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got asset = %+v want %+v", got, want)
	}

	// invalid base58 asset id
	_, err = AssetByID(ctx, "invalid-base58-id")
	if errors.Root(err) != ErrBadAsset {
		t.Errorf("got error = %v want %v", errors.Root(err), ErrBadAsset)
	}

	// missing asset id
	_, err = AssetByID(ctx, "AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("got error = %v want %v", errors.Root(err), pg.ErrUserInputNotFound)
	}
}
