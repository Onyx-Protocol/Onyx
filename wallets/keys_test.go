package wallets

import (
	"chain/database/pg"
	"reflect"
	"testing"
)

func TestKeyIndexSQL(t *testing.T) {
	cases := []struct {
		n    int64
		want []uint32
	}{
		{1, []uint32{0, 1}},
		{2, []uint32{0, 2}},
		{0x80000000, []uint32{1, 0}},
		{0x80000001, []uint32{1, 1}},
		{0x100000000, []uint32{2, 0}},
	}

	for _, test := range cases {
		var got []uint32

		err := db.QueryRow(`SELECT key_index($1)`, test.n).Scan((*pg.Uint32s)(&got))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("key_index(%d) = %v want %v", test.n, got, test.want)
		}
	}
}
