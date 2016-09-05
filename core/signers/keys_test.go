package signers

import (
	"context"
	"reflect"
	"testing"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestKeyIndexSQL(t *testing.T) {
	pairs := []struct {
		encoded int64
		decoded []uint32
	}{
		{1, []uint32{0, 1}},
		{2, []uint32{0, 2}},
		{0x80000000, []uint32{1, 0}},
		{0x80000001, []uint32{1, 1}},
		{0x100000000, []uint32{2, 0}},
	}

	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	for _, pair := range pairs {
		var got []uint32

		err := pg.QueryRow(ctx, `SELECT key_index($1)`, pair.encoded).Scan((*pg.Uint32s)(&got))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if !reflect.DeepEqual(got, pair.decoded) {
			t.Errorf("key_index(%d) = %v want %v", pair.encoded, got, pair.decoded)
		}

		got = keyIndex(pair.encoded)
		if !reflect.DeepEqual(got, pair.decoded) {
			t.Errorf("KeyIndex(%d) = %v want %v", pair.encoded, got, pair.decoded)
		}

		var got2 int64

		err = pg.QueryRow(ctx, `SELECT to_key_index($1::int[])`, pg.Uint32s(pair.decoded)).Scan(&got2)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if got2 != pair.encoded {
			t.Errorf("to_key_index(%v) = %d want %d", pair.decoded, got, pair.encoded)
		}
	}
}

func keyIndex(n int64) []uint32 {
	index := make([]uint32, 2)
	index[0] = uint32(n >> 31)
	index[1] = uint32(n & 0x7fffffff)
	return index
}
