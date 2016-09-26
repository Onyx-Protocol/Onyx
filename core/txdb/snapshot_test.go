package txdb

import (
	"context"
	"reflect"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/state"
)

type pair struct {
	key  string
	hash bc.Hash
}

func TestReadWriteStateSnapshot(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	snapshot := state.Empty()
	changes := []struct {
		inserts          []pair
		deletes          []string
		newIssuances     map[bc.Hash]uint64
		deletedIssuances []bc.Hash
	}{
		{ // add a single k/v pair
			inserts: []pair{
				{
					key:  "sup",
					hash: bc.Hash{0x01},
				},
			},
			newIssuances: map[bc.Hash]uint64{
				bc.Hash{0x01}: 1000,
			},
		},
		{ // empty changeset
		},
		{ // add two pairs
			inserts: []pair{
				{
					key:  "sup",
					hash: bc.Hash{0x02},
				},
				{
					key:  "dup2",
					hash: bc.Hash{0x03},
				},
			},
			newIssuances: map[bc.Hash]uint64{
				bc.Hash{0x02}: 2000,
			},
		},
		{ // delete one pair
			deletes:          []string{"sup"},
			deletedIssuances: []bc.Hash{bc.Hash{0x02}},
		},
		{ // insert and delete at the same time
			inserts: []pair{
				{
					key:  "hello",
					hash: bc.Hash{0x04},
				},
			},
			deletes: []string{"hello"},
		},
	}

	for i, changeset := range changes {
		t.Logf("Applying changeset %d\n", i)

		for _, insert := range changeset.inserts {
			err := snapshot.Tree.Insert([]byte(insert.key), insert.hash)
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, key := range changeset.deletes {
			err := snapshot.Tree.Delete([]byte(key))
			if err != nil {
				t.Fatal(err)
			}
		}

		err := storeStateSnapshot(ctx, dbtx, snapshot, uint64(i))
		if err != nil {
			t.Fatalf("Error writing state snapshot to db: %s\n", err)
		}

		loadedSnapshot, height, err := getStateSnapshot(ctx, dbtx)
		if err != nil {
			t.Fatalf("Error reading state snapshot from db: %s\n", err)
		}

		if height != uint64(i) {
			t.Fatalf("%d: state snapshot height got=%d want=%d", i, height, uint64(i))
		}
		if loadedSnapshot.Tree.RootHash() != snapshot.Tree.RootHash() {
			t.Fatalf("%d: Wrote %s to db, read %s from db\n", i, snapshot.Tree.RootHash(), loadedSnapshot.Tree.RootHash())
		}
		if !reflect.DeepEqual(loadedSnapshot.Issuances, snapshot.Issuances) {
			t.Fatalf("%d: Wrote %#v issuances to db, read %#v from db\n", i, snapshot.Issuances, loadedSnapshot.Issuances)
		}
		snapshot = loadedSnapshot
	}
}
