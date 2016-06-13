package txdb

import (
	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/database/pg/pgtest"
	"testing"

	"golang.org/x/net/context"
)

type pair struct {
	key  string
	hash bc.Hash
}

func TestReadWriteStateTree(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	tree := patricia.NewTree(nil)
	changes := []struct {
		inserts []pair
		deletes []string
	}{
		{ // empty changeset
		},
		{ // add a single k/v pair
			inserts: []pair{
				{
					key:  "sup",
					hash: bc.Hash{0x01},
				},
			},
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
		},
		{ // delete one pair
			deletes: []string{"sup"},
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
			err := tree.Insert([]byte(insert.key), patricia.HashValuer(insert.hash))
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, key := range changeset.deletes {
			err := tree.Delete([]byte(key))
			if err != nil {
				t.Fatal(err)
			}
		}

		err := writeStateTree(ctx, dbtx, tree)
		if err != nil {
			t.Fatalf("Error writing state tree to db: %s\n", err)
		}

		newTree, err := stateTree(ctx, dbtx)
		if err != nil {
			t.Fatalf("Error reading state tree from db: %s\n", err)
		}
		if newTree.RootHash() != tree.RootHash() {
			t.Fatalf("%d: Wrote %s to db, read %s from db\n", i, tree.RootHash(), newTree.RootHash())
		}
		tree = newTree
	}

}
