package confidentiality

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"testing"

	"chain-stealth/database/pg/pgtest"
	"chain-stealth/protocol/bc"
)

func TestStoreAndGetKeys(t *testing.T) {
	ctx := context.Background()
	s := &Storage{DB: pgtest.NewTx(t)}

	err := s.StoreKeys(ctx, []*Key{
		{ControlProgram: []byte{0x01}, Key: [32]byte{0xbe, 0xef}},
		{ControlProgram: []byte{0x02}, Key: [32]byte{0xc0, 0x01}},
		{ControlProgram: []byte{0x03}, Key: [32]byte{0xca, 0xfe}},
	})
	if err != nil {
		t.Fatal(err)
	}

	keys, err := s.GetKeys(ctx, [][]byte{[]byte{0x01}, []byte{0x03}})
	if err != nil {
		t.Fatal(err)
	}

	sort.Sort(byKey(keys))
	if !reflect.DeepEqual(keys, []*Key{
		{ControlProgram: []byte{0x01}, Key: [32]byte{0xbe, 0xef}},
		{ControlProgram: []byte{0x03}, Key: [32]byte{0xca, 0xfe}},
	}) {
		t.Errorf("got %#v", keys)
	}
}

func TestRecordAndLookupIssuances(t *testing.T) {
	ctx := context.Background()
	s := &Storage{DB: pgtest.NewTx(t)}

	issuances := []struct {
		assetID bc.AssetID
		nonce   []byte
		amount  uint64
	}{
		{
			assetID: [32]byte{0x01},
			nonce:   []byte{0xde, 0xad, 0xbe, 0xef},
			amount:  1000,
		},
		{
			assetID: [32]byte{0x02},
			nonce:   []byte{0xc0, 0x01, 0xca, 0xfe},
			amount:  50,
		},
		{
			assetID: [32]byte{0x01},
			nonce:   []byte{0xff},
			amount:  300,
		},
	}
	for _, i := range issuances {
		err := s.RecordIssuance(ctx, i.assetID, i.nonce, i.amount)
		if err != nil {
			t.Fatal(err)
		}
	}

	lookups := []struct {
		assetID bc.AssetID
		nonce   []byte
		amount  uint64
		ok      bool
	}{
		{
			assetID: [32]byte{0x01},
			nonce:   []byte{0xc0, 0x01, 0xca, 0xfe},
			ok:      false,
		},
		{
			assetID: [32]byte{0x01},
			nonce:   []byte{0xde, 0xad, 0xbe, 0xef},
			amount:  1000,
			ok:      true,
		},
		{
			assetID: [32]byte{0x02},
			nonce:   []byte{0xc0, 0x01, 0xca, 0xfe},
			amount:  50,
			ok:      true,
		},
		{
			assetID: [32]byte{0x01},
			nonce:   []byte{0xff},
			amount:  300,
			ok:      true,
		},
	}

	for i, l := range lookups {
		amt, ok, err := s.lookupIssuance(ctx, l.assetID, l.nonce)
		if err != nil {
			t.Error(err)
		}
		if ok != l.ok {
			t.Error("lookup ok %d: got %t wanted %t", i, ok, l.ok)
		}
		if l.ok && amt != l.amount {
			t.Error("lookup amount %d: got %d wanted %d", i, amt, l.amount)
		}
	}

}

type byKey []*Key

func (s byKey) Len() int           { return len(s) }
func (s byKey) Less(i, j int) bool { return bytes.Compare(s[i].Key[:], s[j].Key[:]) == -1 }
func (s byKey) Swap(i, j int)      { s[j], s[j] = s[j], s[i] }
