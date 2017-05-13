package patricia

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
	"chain/testutil"
)

func BenchmarkInserts(b *testing.B) {
	const nodes = 10000
	for i := 0; i < b.N; i++ {
		r := rand.New(rand.NewSource(12345))
		tr := new(Tree)
		for j := 0; j < nodes; j++ {
			var h [32]byte
			_, err := r.Read(h[:])
			if err != nil {
				b.Fatal(err)
			}

			err = tr.Insert(h[:])
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkInsertsRootHash(b *testing.B) {
	const nodes = 10000
	for i := 0; i < b.N; i++ {
		r := rand.New(rand.NewSource(12345))
		tr := new(Tree)
		for j := 0; j < nodes; j++ {
			var h [32]byte
			_, err := r.Read(h[:])
			if err != nil {
				b.Fatal(err)
			}

			err = tr.Insert(h[:])
			if err != nil {
				b.Fatal(err)
			}
		}
		tr.RootHash()
	}
}

func TestRootHashBug(t *testing.T) {
	tr := new(Tree)

	err := tr.Insert([]byte{0x94})
	if err != nil {
		t.Fatal(err)
	}
	err = tr.Insert([]byte{0x36})
	if err != nil {
		t.Fatal(err)
	}
	before := tr.RootHash()
	err = tr.Insert([]byte{0xba})
	if err != nil {
		t.Fatal(err)
	}
	if tr.RootHash() == before {
		t.Errorf("before and after root hash is %s", before.String())
	}
}

func TestLeafVsInternalNodes(t *testing.T) {
	tr0 := new(Tree)

	err := tr0.Insert([]byte{0x01})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x02})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x03})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x04})
	if err != nil {
		t.Fatal(err)
	}

	// Force calculation of all the hashes.
	tr0.RootHash()
	t.Logf("first child = %s, %t", tr0.root.children[0].hash, tr0.root.children[0].isLeaf)
	t.Logf("second child = %s, %t", tr0.root.children[1].hash, tr0.root.children[1].isLeaf)

	// Create a second tree using an internal node from tr1.
	tr1 := new(Tree)
	err = tr1.Insert(tr0.root.children[0].hash.Bytes()) // internal node of tr0
	if err != nil {
		t.Fatal(err)
	}
	err = tr1.Insert(tr0.root.children[1].hash.Bytes()) // sibling leaf node of above node ^
	if err != nil {
		t.Fatal(err)
	}

	if tr1.RootHash() == tr0.RootHash() {
		t.Errorf("tr0 and tr1 have matching root hashes: %x", tr1.RootHash().Bytes())
	}
}

func TestRootHashInsertQuickCheck(t *testing.T) {
	tr := new(Tree)

	f := func(b [32]byte) bool {
		before := tr.RootHash()
		err := tr.Insert(b[:])
		if err != nil {
			return false
		}
		return before != tr.RootHash()
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestLookup(t *testing.T) {
	tr := &Tree{
		root: &node{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
	}
	got := lookup(tr.root, bitKey(bits("11111111")))
	if !testutil.DeepEqual(got, tr.root) {
		t.Log("lookup on 1-node tree")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root, 0))
	}

	tr = &Tree{
		root: &node{key: bools("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true},
	}
	got = lookup(tr.root, bitKey(bits("11111111")))
	if got != nil {
		t.Log("lookup nonexistent key on 1-node tree")
		t.Fatalf("got:\n%swant nil", prettyNode(got, 0))
	}

	tr = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11110000")), hashForLeaf(bits("11111111")))),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
			},
		},
	}
	got = lookup(tr.root, bitKey(bits("11110000")))
	if !testutil.DeepEqual(got, tr.root.children[0]) {
		t.Log("lookup root's first child")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[0], 0))
	}

	tr = &Tree{
		root: &node{
			key: bools("1111"),
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111"))),
			)),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111")))),
					children: [2]*node{
						{key: bools("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true},
						{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
					},
				},
			},
		},
	}
	got = lookup(tr.root, bitKey(bits("11111100")))
	if !testutil.DeepEqual(got, tr.root.children[1].children[0]) {
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[1].children[0], 0))
	}
}

func TestContains(t *testing.T) {
	tr := new(Tree)
	tr.Insert(bits("00000011"))
	tr.Insert(bits("00000010"))

	if v := bits("00000011"); !tr.Contains(v) {
		t.Errorf("expected tree to contain %x, but did not", v)
	}
	if v := bits("00000000"); tr.Contains(v) {
		t.Errorf("expected tree to not contain %x, but did", v)
	}
	if v := bits("00000010"); !tr.Contains(v) {
		t.Errorf("expected tree to contain %x, but did not", v)
	}
}

func TestInsert(t *testing.T) {
	tr := new(Tree)

	tr.Insert(bits("11111111"))
	tr.RootHash()
	want := &Tree{
		root: &node{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		log.Printf("want hash? %x", hashForLeaf(bits("11111111")).Bytes())
		t.Log("insert into empty tree")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111111"))
	tr.RootHash()
	want = &Tree{
		root: &node{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("inserting the same key does not modify the tree")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11110000"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11110000")), hashForLeaf(bits("11111111")))),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("different key creates a fork")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111100"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key: bools("1111"),
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111"))),
			)),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111")))),
					children: [2]*node{
						{key: bools("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true},
						{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111110"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key: bools("1111"),
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(
					hashForLeaf(bits("11111100")),
					hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
				),
			)),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{
					key: bools("111111"),
					hash: hashPtr(hashForNonLeaf(
						hashForLeaf(bits("11111100")),
						hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))))),
					children: [2]*node{
						{key: bools("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true},
						{
							key:  bools("1111111"),
							hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111")))),
							children: [2]*node{
								{key: bools("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true},
								{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
							},
						},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("a fork is created for each level of similar key")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111011"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key: bools("1111"),
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(
					hashForLeaf(bits("11111011")),
					hashForNonLeaf(
						hashForLeaf(bits("11111100")),
						hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
					),
				),
			)),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{
					key: bools("11111"),
					hash: hashPtr(hashForNonLeaf(
						hashForLeaf(bits("11111011")),
						hashForNonLeaf(
							hashForLeaf(bits("11111100")),
							hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
						))),
					children: [2]*node{
						{key: bools("11111011"), hash: hashPtr(hashForLeaf(bits("11111011"))), isLeaf: true},
						{
							key: bools("111111"),
							hash: hashPtr(hashForNonLeaf(
								hashForLeaf(bits("11111100")),
								hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
							)),
							children: [2]*node{
								{key: bools("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true},
								{
									key:  bools("1111111"),
									hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111")))),
									children: [2]*node{
										{key: bools("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true},
										{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("compressed branch node is split")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}
}

func TestDelete(t *testing.T) {
	tr := new(Tree)
	tr.root = &node{
		key: bools("1111"),
		hash: hashPtr(hashForNonLeaf(
			hashForLeaf(bits("11110000")),
			hashForNonLeaf(
				hashForLeaf(bits("11111100")),
				hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
			),
		)),
		children: [2]*node{
			{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
			{
				key: bools("111111"),
				hash: hashPtr(hashForNonLeaf(
					hashForLeaf(bits("11111100")),
					hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
				)),
				children: [2]*node{
					{key: bools("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true},
					{
						key:  bools("1111111"),
						hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111")))),
						children: [2]*node{
							{key: bools("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true},
							{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
						},
					},
				},
			},
		},
	}

	tr.Delete(bits("11111110"))
	tr.RootHash()
	want := &Tree{
		root: &node{
			key: bools("1111"),
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111"))),
			)),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111")))),
					children: [2]*node{
						{key: bools("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true},
						{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111100"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11110000")), hashForLeaf(bits("11111111")))),
			children: [2]*node{
				{key: bools("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true},
				{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110011")) // nonexistent value
	tr.RootHash()
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110000"))
	tr.RootHash()
	want = &Tree{
		root: &node{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111111"))
	tr.RootHash()
	want = &Tree{}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}
}

func TestDeletePrefix(t *testing.T) {
	root := &node{
		key:  bools("111111"),
		hash: hashPtr(hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111")))),
		children: [2]*node{
			{key: bools("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true},
			{key: bools("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true},
		},
	}

	got := delete(root, bools("111111"))
	got.calcHash()
	if !testutil.DeepEqual(got, root) {
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(root, 0))
	}
}

func TestBoolKey(t *testing.T) {
	cases := []struct {
		b []byte
		w []uint8
	}{{
		b: nil,
		w: []uint8{},
	}, {
		b: []byte{0x8f},
		w: []uint8{1, 0, 0, 0, 1, 1, 1, 1},
	}, {
		b: []byte{0x81},
		w: []uint8{1, 0, 0, 0, 0, 0, 0, 1},
	}}

	for _, c := range cases {
		g := bitKey(c.b)

		if !testutil.DeepEqual(g, c.w) {
			t.Errorf("Key(0x%x) = %v want %v", c.b, g, c.w)
		}
	}
}

func TestByteKey(t *testing.T) {
	cases := []struct {
		b []uint8
		w []byte
	}{{
		b: []uint8{},
		w: []byte{},
	}, {
		b: []uint8{1, 0, 0, 0, 1, 1, 1, 1},
		w: []byte{0x8f},
	}, {
		b: []uint8{1, 0, 0, 0, 0, 0, 0, 1},
		w: []byte{0x81},
	}, {
		b: []uint8{1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 1, 1},
		w: []byte{0x81, 0x8f},
	}}

	for _, c := range cases {
		g := byteKey(c.b)

		if !testutil.DeepEqual(g, c.w) {
			t.Errorf("byteKey(%#v) = %x want %x", c.b, g, c.w)
		}
	}
}

func pretty(t *Tree) string {
	if t.root == nil {
		return ""
	}
	return prettyNode(t.root, 0)
}

func prettyNode(n *node, depth int) string {
	prettyStr := strings.Repeat("  ", depth)
	if n == nil {
		prettyStr += "nil\n"
		return prettyStr
	}
	var b int
	if len(n.key) > 31*8 {
		b = 31 * 8
	}
	prettyStr += fmt.Sprintf("key=%+v", n.key[b:])
	if n.hash != nil {
		prettyStr += fmt.Sprintf(" hash=%+v", n.hash)
	}
	prettyStr += "\n"

	for _, c := range n.children {
		if c != nil {
			prettyStr += prettyNode(c, depth+1)
		}
	}

	return prettyStr
}

func bits(lit string) []byte {
	var b [31]byte
	n, _ := strconv.ParseUint(lit, 2, 8)
	return append(b[:], byte(n))
}

func bools(lit string) []uint8 {
	b := bitKey(bits(lit))
	return append(b[:31*8], b[32*8-len(lit):]...)
}

func hashForLeaf(item []byte) bc.Hash {
	return bc.NewHash(sha3.Sum256(append([]byte{0x00}, item...)))
}

func hashForNonLeaf(a, b bc.Hash) bc.Hash {
	d := []byte{0x01}
	d = append(d, a.Bytes()...)
	d = append(d, b.Bytes()...)
	return bc.NewHash(sha3.Sum256(d))
}

func hashPtr(h bc.Hash) *bc.Hash {
	return &h
}
