package txdb

import (
	"context"
	"math/rand"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/testutil"
)

func TestReadWriteStateSnapshot(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	snapshot := state.Empty()
	changes := []struct {
		inserts          []bc.Hash
		deletes          []bc.Hash
		lookups          []bc.Hash
		newIssuances     map[bc.Hash]uint64
		deletedIssuances []bc.Hash
	}{
		{ // add a single hash
			inserts: []bc.Hash{{0x01}},
			newIssuances: map[bc.Hash]uint64{
				bc.Hash{0x01}: 1000,
			},
		},
		{ // empty changeset
			lookups: []bc.Hash{{0x01}},
		},
		{ // add two new hashes
			inserts: []bc.Hash{
				{0x02},
				{0x03},
			},
			lookups: []bc.Hash{
				{0x02},
				{0x03},
			},
			newIssuances: map[bc.Hash]uint64{
				bc.Hash{0x02}: 2000,
			},
		},
		{ // delete one hash
			deletes:          []bc.Hash{{0x01}},
			deletedIssuances: []bc.Hash{{0x02}},
		},
		{ // insert and delete at the same time
			inserts: []bc.Hash{{0x04}},
			deletes: []bc.Hash{{0x04}},
		},
	}

	for i, changeset := range changes {
		t.Logf("Applying changeset %d\n", i)

		for _, insert := range changeset.inserts {
			err := snapshot.Tree.Insert(insert[:], insert[:])
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, key := range changeset.deletes {
			err := snapshot.Tree.Delete(key[:])
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

		for _, lookup := range changeset.lookups {
			if !snapshot.Tree.Contains(lookup[:], lookup[:]) {
				t.Errorf("Lookup(%s, %s) = false, want true", lookup, lookup)
			}
		}

		if height != uint64(i) {
			t.Fatalf("%d: state snapshot height got=%d want=%d", i, height, uint64(i))
		}
		if loadedSnapshot.Tree.RootHash() != snapshot.Tree.RootHash() {
			t.Fatalf("%d: Wrote %s to db, read %s from db\n", i, snapshot.Tree.RootHash(), loadedSnapshot.Tree.RootHash())
		}
		if !testutil.DeepEqual(loadedSnapshot.Issuances, snapshot.Issuances) {
			t.Fatalf("%d: Wrote %#v issuances to db, read %#v from db\n", i, snapshot.Issuances, loadedSnapshot.Issuances)
		}
		snapshot = loadedSnapshot
	}
}

func BenchmarkStoreSnapshot100(b *testing.B) {
	benchmarkStoreSnapshot(100, 100, b)
}

func BenchmarkStoreSnapshot1000(b *testing.B) {
	benchmarkStoreSnapshot(1000, 1000, b)
}

func BenchmarkStoreSnapshot10000(b *testing.B) {
	benchmarkStoreSnapshot(10000, 10000, b)
}

func benchmarkStoreSnapshot(nodes, issuances int, b *testing.B) {
	b.StopTimer()

	// Generate a snapshot with a large number of existing patricia
	// tree nodes and issuances.
	r := rand.New(rand.NewSource(12345))
	db := pgtest.NewTx(b)
	ctx := context.Background()

	snapshot := state.Empty()
	for i := 0; i < nodes; i++ {
		var h [32]byte
		_, err := r.Read(h[:])
		if err != nil {
			b.Fatal(err)
		}

		err = snapshot.Tree.Insert(h[:], h[:])
		if err != nil {
			b.Fatal(err)
		}
	}

	for i := 0; i < issuances; i++ {
		var h bc.Hash
		_, err := r.Read(h[:])
		if err != nil {
			b.Fatal(err)
		}

		snapshot.Issuances[h] = uint64(r.Int63())
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := storeStateSnapshot(ctx, db, snapshot, uint64(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}
