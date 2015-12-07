package auditor

import (
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	chainlog "chain/log"
)

func init() {
	chainlog.SetOutput(ioutil.Discard)

	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db := pgtest.Open(u, "auditortest", "../appdb/schema.sql")
	err := appdb.Init(db)
	if err != nil {
		log.Fatal(err)
	}
}

// Establish a context object with a new db transaction in which to
// run the given callback function.
func withContext(tb testing.TB, sql string, fn func(context.Context)) {
	var dbtx pg.Tx
	if sql == "" {
		dbtx = pgtest.TxWithSQL(tb)
	} else {
		dbtx = pgtest.TxWithSQL(tb, sql)
	}
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	fn(ctx)
}

func mustParseHash(str string) bc.Hash {
	hash, err := bc.ParseHash(str)
	if err != nil {
		panic(err)
	}
	return hash
}

func mustDecodeHex(str string) []byte {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return bytes
}

func TestListBlocks(t *testing.T) {
	const fix = `
		INSERT INTO blocks(block_hash, height, data)
		VALUES(
			'b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e174',
			1,
			decode('0000000100000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000064000001000000010000000000000000000003747831', 'hex')
		), (
			'40c17dd9b835108e04e080a3033ea6f84681afbeec28305259ed0b519daf6f61',
			2,
			decode('000000010000000000000002b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e174000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000069000002000000010000000000000000000003747832000000010000000000000000000003747833', 'hex')
		);
	`
	withContext(t, fix, func(ctx context.Context) {
		cases := []struct {
			prev     string
			limit    int
			want     []ListBlocksItem
			wantLast string
		}{{
			prev:  "",
			limit: 50,
			want: []ListBlocksItem{{
				ID:      mustParseHash("40c17dd9b835108e04e080a3033ea6f84681afbeec28305259ed0b519daf6f61"),
				Height:  2,
				Time:    time.Unix(105, 0).UTC(),
				TxCount: 2,
			}, {
				ID:      mustParseHash("b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e174"),
				Height:  1,
				Time:    time.Unix(100, 0).UTC(),
				TxCount: 1,
			}},
			wantLast: "",
		}, {
			prev:  "2",
			limit: 50,
			want: []ListBlocksItem{{
				ID:      mustParseHash("b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e174"),
				Height:  1,
				Time:    time.Unix(100, 0).UTC(),
				TxCount: 1,
			}},
			wantLast: "",
		}, {
			prev:  "",
			limit: 1,
			want: []ListBlocksItem{{
				ID:      mustParseHash("40c17dd9b835108e04e080a3033ea6f84681afbeec28305259ed0b519daf6f61"),
				Height:  2,
				Time:    time.Unix(105, 0).UTC(),
				TxCount: 2,
			}},
			wantLast: "2",
		}, {
			prev:     "1",
			limit:    50,
			want:     nil,
			wantLast: "",
		}}
		for _, c := range cases {
			got, gotLast, err := ListBlocks(ctx, c.prev, c.limit)
			if err != nil {
				t.Errorf("ListBlocks(%v, %v) unexpected err = %q", c.prev, c.limit, err)
				continue
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got ListBlocks(%v, %v) = %+v want %+v", c.prev, c.limit, got, c.want)
			}
			if gotLast != c.wantLast {
				t.Errorf("got ListBlocks(%v, %v) last = %q want %q", c.prev, c.limit, gotLast, c.wantLast)
			}
		}
	})
}

func TestGetBlockSummary(t *testing.T) {
	const fix = `
		INSERT INTO blocks(block_hash, height, data)
		VALUES(
			'40c17dd9b835108e04e080a3033ea6f84681afbeec28305259ed0b519daf6f61',
			2,
			decode('000000010000000000000002b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e174000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000069000002000000010000000000000000000003747832000000010000000000000000000003747833', 'hex')
		);
	`
	withContext(t, fix, func(ctx context.Context) {
		got, err := GetBlockSummary(ctx, "40c17dd9b835108e04e080a3033ea6f84681afbeec28305259ed0b519daf6f61")
		if err != nil {
			t.Fatal(err)
		}
		want := &BlockSummary{
			ID:      mustParseHash("40c17dd9b835108e04e080a3033ea6f84681afbeec28305259ed0b519daf6f61"),
			Height:  2,
			Time:    time.Unix(105, 0).UTC(),
			TxCount: 2,
			TxIDs: []bc.Hash{
				mustParseHash("44820f8498ba868d6a943955694451d7672bed72d1330ffeaab9bed0dae78b87"),
				mustParseHash("64738403957ac8347d95a589eeb733f4c5c2d71f83a457ac44f1638eb4d4286b"),
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got block header:\n\t%+v\nwant:\n\t%+v", got, want)
		}
	})
}

func TestGetTxIssuance(t *testing.T) {
	tx := &bc.Tx{
		Inputs: []*bc.TxInput{{
			Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
			Metadata:        []byte(`{"a":"b"}`),
			AssetDefinition: []byte(`{"c":"d"}`),
		}},
		Outputs: []*bc.TxOutput{{
			AssetID:  bc.AssetID([32]byte{0}),
			Value:    5,
			Metadata: []byte{2},
			Script:   mustDecodeHex("a914488f20d75535a8f408f47d954cbbb319482ca68987"), // 38Jg35n4ne2C6rDLCYM94odqgWG9QZW1SW
		}, {
			AssetID: bc.AssetID([32]byte{0}),
			Value:   6,
			Script:  mustDecodeHex("a91430819f1955f747220bb247df8a989e36a432733487"), // 367Ve1Xkgwwiu9rmm9bJEnB91ZFnn79M1P
		}},
		Metadata: []byte{0},
	}

	withContext(t, "", func(ctx context.Context) {
		err := txdb.InsertTx(ctx, tx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		got, err := GetTx(ctx, tx.Hash().String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		want := &Tx{
			ID:       tx.Hash(),
			BlockID:  nil,
			Metadata: []byte{0},
			Inputs: []*TxInput{{
				Type:     "issuance",
				AssetID:  bc.AssetID([32]byte{0}),
				Amount:   11,
				Metadata: []byte(`{"a":"b"}`),
				AssetDef: map[string]interface{}{"c": "d"},
			}},
			Outputs: []*TxOutput{{
				AssetID:  bc.AssetID([32]byte{0}),
				Amount:   5,
				Address:  "38Jg35n4ne2C6rDLCYM94odqgWG9QZW1SW",
				Metadata: []byte{2},
			}, {
				AssetID: bc.AssetID([32]byte{0}),
				Amount:  6,
				Address: "367Ve1Xkgwwiu9rmm9bJEnB91ZFnn79M1P",
			}},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
		}
	})
}

func TestGetTxTransfer(t *testing.T) {
	prevTxs := []*bc.Tx{{
		Outputs: []*bc.TxOutput{{
			AssetID: bc.AssetID([32]byte{1}),
			Value:   5,
		}},
	}, {
		Outputs: []*bc.TxOutput{{}, {
			AssetID: bc.AssetID([32]byte{2}),
			Value:   6,
		}},
	}}
	tx := &bc.Tx{
		Inputs: []*bc.TxInput{{
			Previous: bc.Outpoint{Hash: prevTxs[0].Hash(), Index: 0},
		}, {
			Previous: bc.Outpoint{Hash: prevTxs[1].Hash(), Index: 1},
		}},
		Outputs: []*bc.TxOutput{{
			AssetID: bc.AssetID([32]byte{1}),
			Value:   5,
			Script:  mustDecodeHex("a914488f20d75535a8f408f47d954cbbb319482ca68987"), // 38Jg35n4ne2C6rDLCYM94odqgWG9QZW1SW
		}, {
			AssetID: bc.AssetID([32]byte{2}),
			Value:   6,
			Script:  mustDecodeHex("a91430819f1955f747220bb247df8a989e36a432733487"), // 367Ve1Xkgwwiu9rmm9bJEnB91ZFnn79M1P
		}},
	}

	now := time.Now().UTC().Truncate(time.Second)
	blk := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Height:    1,
			Timestamp: uint64(now.Unix()),
		},
		Transactions: append(prevTxs, tx),
	}

	withContext(t, "", func(ctx context.Context) {
		const q = `INSERT INTO txs (tx_hash, data) VALUES($1, $2)`
		err := txdb.InsertBlock(ctx, blk)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		got, err := GetTx(ctx, tx.Hash().String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		var blkHash = blk.Hash()

		h0, h1 := prevTxs[0].Hash(), prevTxs[1].Hash()

		want := &Tx{
			ID:          tx.Hash(),
			BlockID:     &blkHash,
			BlockHeight: 1,
			BlockTime:   now,
			Inputs: []*TxInput{{
				Type:    "transfer",
				AssetID: bc.AssetID([32]byte{1}),
				Amount:  5,
				TxID:    &h0,
				TxOut:   0,
			}, {
				Type:    "transfer",
				AssetID: bc.AssetID([32]byte{2}),
				Amount:  6,
				TxID:    &h1,
				TxOut:   1,
			}},
			Outputs: []*TxOutput{{
				AssetID: bc.AssetID([32]byte{1}),
				Amount:  5,
				Address: "38Jg35n4ne2C6rDLCYM94odqgWG9QZW1SW",
			}, {
				AssetID: bc.AssetID([32]byte{2}),
				Amount:  6,
				Address: "367Ve1Xkgwwiu9rmm9bJEnB91ZFnn79M1P",
			}},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
		}
	})
}

func TestGetAsset(t *testing.T) {
	const fix = `
		INSERT INTO projects (id, name) VALUES('proj-1', 'foo');
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
			VALUES ('inode-1', 'proj-1', 'bar', '{}');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, label, issuance_script,
				issued_pool, issued_confirmed)
			VALUES ('asset-1', 'inode-1', 0, '', 'baz', '', 5, 6);
		INSERT INTO asset_definition_pointers (asset_id, asset_definition_hash)
			VALUES ('asset-1', 'hash-1');
		INSERT INTO asset_definitions (hash, definition)
			VALUES ('hash-1', '{"a":"b"}'::bytea);
	`
	withContext(t, fix, func(ctx context.Context) {
		got, err := GetAsset(ctx, "asset-1")
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		want := &Asset{
			ID:            "asset-1",
			DefinitionPtr: "hash-1",
			Definition:    map[string]interface{}{"a": "b"},
			Circulation:   appdb.AssetCirculation{Total: 11, Confirmed: 6},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
		}

		_, err = GetAsset(ctx, "nonexistent")
		if errors.Root(err) != pg.ErrUserInputNotFound {
			t.Errorf("got err = %q want %q", errors.Root(err), pg.ErrUserInputNotFound)
		}
	})
}
