package generator_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "chain/core/generator"
	"chain/core/txdb"
	"chain/cos"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
)

func hashForFixture(h bc.Hash) string {
	return fmt.Sprintf("decode('%s', 'hex')", hex.EncodeToString(h[:]))
}

func blockForFixture(b *bc.Block) string {
	buf := new(bytes.Buffer)
	_, err := b.WriteTo(buf)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("decode('%s', 'hex')", hex.EncodeToString(buf.Bytes()))
}

func txForFixture(tx *bc.TxData) string {
	data, err := tx.MarshalText()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("decode('%s', 'hex')", hex.EncodeToString(data))
}

func TestGetSummary(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(pg.FromContext(ctx).(*sql.DB))
	fc, err := cos.NewFC(ctx, store, pool, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	generator := Generator{
		Config: Config{
			BlockPeriod: time.Second,
			FC:          fc,
		},
	}
	if err != nil {
		t.Fatal(err)
	}

	b1 := bc.Block{BlockHeader: bc.BlockHeader{Height: 1}}
	b2 := bc.Block{BlockHeader: bc.BlockHeader{Height: 2}}

	t0 := bc.TxData{Metadata: []byte{0}}
	t1 := bc.TxData{Metadata: []byte{1}}
	t2 := bc.TxData{Metadata: []byte{2}}
	t3 := bc.TxData{Metadata: []byte{3}}
	t4 := bc.TxData{Metadata: []byte{4}}

	pgtest.Exec(ctx, t, `
		INSERT INTO projects
			(id, name)
		VALUES
			('proj-id-0', 'proj-name-0'),
			('proj-id-other', 'proj-name-other');

		INSERT INTO issuer_nodes
			(id, project_id, key_index, label, keyset)
		VALUES
			('in-id-0', 'proj-id-0', 3, 'in-label-0', '{}'),
			('in-id-1', 'proj-id-0', 4, 'in-label-1', '{}'),
			('mn-id-other', 'proj-id-other', 5, 'in-label-other', '{}');

		INSERT INTO blocks
			(block_hash, height, data, header)
		VALUES
			(`+hashForFixture(b1.Hash())+`, `+strconv.Itoa(int(b1.Height))+`, `+blockForFixture(&b1)+`, ''),
			(`+hashForFixture(b2.Hash())+`, `+strconv.Itoa(int(b2.Height))+`, `+blockForFixture(&b2)+`, '');

		INSERT INTO blocks_txs
			(block_hash, tx_hash, block_height, block_pos)
		VALUES
			(`+hashForFixture(b1.Hash())+`, `+hashForFixture(t0.Hash())+`, 1, 0),
			(`+hashForFixture(b2.Hash())+`, `+hashForFixture(t1.Hash())+`, 1, 1),
			(`+hashForFixture(b2.Hash())+`, `+hashForFixture(t2.Hash())+`, 1, 2);

		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			(`+hashForFixture(t3.Hash())+`, `+txForFixture(&t3)+`),
			(`+hashForFixture(t4.Hash())+`, `+txForFixture(&t4)+`);
	`)

	want := &Summary{
		BlockFreqMs: 1000,
		BlockCount:  2,
		TransactionCount: TxCount{
			Confirmed:   3,
			Unconfirmed: 2,
		},
		Permissions: NodePerms{
			ManagerNodes: []NodePermStatus{},
			IssuerNodes:  []NodePermStatus{},
			AuditorNodes: []NodePermStatus{
				{"audnode-proj-id-0", "Auditor Node for proj-id-0", true},
			},
		},
	}

	got, err := generator.GetSummary(ctx, store, pool, "proj-id-0")
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("summary:\ngot:  %v\nwant: %v", *got, *want)
	}
}
