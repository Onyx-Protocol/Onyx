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

func TestReadWriteStateSnapshotNonceSet(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	snapshot := state.Empty()
	snapshot.Nonces[bc.NewHash([32]byte{0x01})] = 10
	snapshot.Nonces[bc.NewHash([32]byte{0x02})] = 10
	snapshot.Nonces[bc.NewHash([32]byte{0x03})] = 45
	err := storeStateSnapshot(ctx, dbtx, snapshot, 200)
	if err != nil {
		t.Fatalf("Error writing state snapshot to db: %s\n", err)
	}
	got, _, err := getStateSnapshot(ctx, dbtx)
	if err != nil {
		t.Fatalf("Error reading state snapshot from db: %s\n", err)
	}
	want := map[bc.Hash]uint64{
		bc.NewHash([32]byte{0x01}): 10,
		bc.NewHash([32]byte{0x02}): 10,
		bc.NewHash([32]byte{0x03}): 45,
	}
	if !testutil.DeepEqual(got.Nonces, want) {
		t.Errorf("storing and loading snapshot nonce memory, got %#v, want %#v", got.Nonces, want)
	}
}

func TestReadWriteStateSnapshot(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	snapshot := state.Empty()
	changes := []struct {
		inserts       []bc.Hash
		deletes       []bc.Hash
		lookups       []bc.Hash
		newNonces     map[bc.Hash]uint64
		deletedNonces []bc.Hash
	}{
		{ // add a single hash
			inserts: []bc.Hash{bc.NewHash([32]byte{0x01})},
			newNonces: map[bc.Hash]uint64{
				bc.NewHash([32]byte{0x01}): 1000,
			},
		},
		{ // empty changeset
			lookups: []bc.Hash{bc.NewHash([32]byte{0x01})},
		},
		{ // add two new hashes
			inserts: []bc.Hash{
				bc.NewHash([32]byte{0x02}),
				bc.NewHash([32]byte{0x03}),
			},
			lookups: []bc.Hash{
				bc.NewHash([32]byte{0x02}),
				bc.NewHash([32]byte{0x03}),
			},
			newNonces: map[bc.Hash]uint64{
				bc.NewHash([32]byte{0x02}): 2000,
			},
		},
		{ // delete one hash
			deletes:       []bc.Hash{bc.NewHash([32]byte{0x01})},
			deletedNonces: []bc.Hash{bc.NewHash([32]byte{0x02})},
		},
		{ // insert and delete at the same time
			inserts: []bc.Hash{bc.NewHash([32]byte{0x04})},
			deletes: []bc.Hash{bc.NewHash([32]byte{0x04})},
		},
	}

	for i, changeset := range changes {
		t.Logf("Applying changeset %d\n", i)

		for _, insert := range changeset.inserts {
			err := snapshot.Tree.Insert(insert.Bytes())
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, key := range changeset.deletes {
			snapshot.Tree.Delete(key.Bytes())
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
			if !snapshot.Tree.Contains(lookup.Bytes()) {
				t.Errorf("Lookup(%s, %s) = false, want true", lookup.String(), lookup.String())
			}
		}

		if height != uint64(i) {
			t.Fatalf("%d: state snapshot height got=%d want=%d", i, height, uint64(i))
		}
		if loadedSnapshot.Tree.RootHash() != snapshot.Tree.RootHash() {
			t.Fatalf("%d: Wrote %x to db, read %x from db\n", i, snapshot.Tree.RootHash().Bytes(), loadedSnapshot.Tree.RootHash().Bytes())
		}
		if !testutil.DeepEqual(loadedSnapshot.Nonces, snapshot.Nonces) {
			t.Fatalf("%d: Wrote %#v nonces to db, read %#v from db\n", i, snapshot.Nonces, loadedSnapshot.Nonces)
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

func benchmarkStoreSnapshot(nodes, nonces int, b *testing.B) {
	b.StopTimer()

	// Generate a snapshot with a large number of existing patricia
	// tree nodes and nonces.
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

		err = snapshot.Tree.Insert(h[:])
		if err != nil {
			b.Fatal(err)
		}
	}

	for i := 0; i < nonces; i++ {
		var h bc.Hash
		_, err := h.ReadFrom(r)
		if err != nil {
			b.Fatal(err)
		}

		snapshot.Nonces[h] = uint64(r.Int63())
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := storeStateSnapshot(ctx, db, snapshot, uint64(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}
