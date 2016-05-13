package generator_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	. "chain/core/generator"
	"chain/core/txdb"
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
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB))

	b0 := bc.Block{BlockHeader: bc.BlockHeader{Height: 0}}
	b1 := bc.Block{BlockHeader: bc.BlockHeader{Height: 1}}

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

		INSERT INTO manager_nodes
			(id, project_id, key_index, label)
		VALUES
			('mn-id-0', 'proj-id-0', 0, 'mn-label-0'),
			('mn-id-1', 'proj-id-0', 1, 'mn-label-1'),
			('mn-id-other', 'proj-id-other', 2, 'mn-label-other');

		INSERT INTO issuer_nodes
			(id, project_id, key_index, label, keyset)
		VALUES
			('in-id-0', 'proj-id-0', 3, 'in-label-0', '{}'),
			('in-id-1', 'proj-id-0', 4, 'in-label-1', '{}'),
			('mn-id-other', 'proj-id-other', 5, 'in-label-other', '{}');

		INSERT INTO blocks
			(block_hash, height, data, header)
		VALUES
			(`+hashForFixture(b0.Hash())+`, `+strconv.Itoa(int(b0.Height))+`, `+blockForFixture(&b0)+`, ''),
			(`+hashForFixture(b1.Hash())+`, `+strconv.Itoa(int(b1.Height))+`, `+blockForFixture(&b1)+`, '');

		INSERT INTO blocks_txs
			(block_hash, tx_hash, block_height, block_pos)
		VALUES
			(`+hashForFixture(b0.Hash())+`, `+hashForFixture(t0.Hash())+`, 1, 0),
			(`+hashForFixture(b1.Hash())+`, `+hashForFixture(t1.Hash())+`, 1, 1),
			(`+hashForFixture(b1.Hash())+`, `+hashForFixture(t2.Hash())+`, 1, 2);

		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			(`+hashForFixture(t3.Hash())+`, `+txForFixture(&t3)+`),
			(`+hashForFixture(t4.Hash())+`, `+txForFixture(&t4)+`);
	`)

	want := &Summary{
		BlockFreqMs: 0,
		BlockCount:  2,
		TransactionCount: TxCount{
			Confirmed:   3,
			Unconfirmed: 2,
		},
		Permissions: NodePerms{
			ManagerNodes: []NodePermStatus{
				{"mn-id-0", "mn-label-0", true},
				{"mn-id-1", "mn-label-1", true},
			},
			IssuerNodes: []NodePermStatus{
				{"in-id-0", "in-label-0", true},
				{"in-id-1", "in-label-1", true},
			},
			AuditorNodes: []NodePermStatus{
				{"audnode-proj-id-0", "Auditor Node for proj-id-0", true},
			},
		},
	}

	got, err := GetSummary(ctx, store, "proj-id-0")
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("summary:\ngot:  %v\nwant: %v", *got, *want)
	}
}
