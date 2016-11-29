package patricia

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
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

			err = tr.Insert(h[:], h[:])
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

			err = tr.Insert(h[:], h[:])
			if err != nil {
				b.Fatal(err)
			}
		}
		tr.RootHash()
	}
}

func TestRootHashBug(t *testing.T) {
	tr := new(Tree)

	err := tr.Insert([]byte{0x94}, []byte{0x01})
	if err != nil {
		t.Fatal(err)
	}
	err = tr.Insert([]byte{0x36}, []byte{0x02})
	if err != nil {
		t.Fatal(err)
	}
	before := tr.RootHash()
	err = tr.Insert([]byte{0xba}, []byte{0x03})
	if err != nil {
		t.Fatal(err)
	}
	if tr.RootHash() == before {
		t.Errorf("before and after root hash is %s", before)
	}
}

func TestLeafVsInternalNodes(t *testing.T) {
	tr0 := new(Tree)

	err := tr0.Insert([]byte{0x01}, []byte{0x01})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x02}, []byte{0x02})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x03}, []byte{0x03})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x04}, []byte{0x04})
	if err != nil {
		t.Fatal(err)
	}

	// Create a second tree using an internal node from tr1.
	tr1 := new(Tree)
	err = tr1.Insert([]byte{0x02}, mustDecodeHash("82b08f644c16985d2d9961b4104cc4bf4ba2be6bb5c3d0df2ecb94149f212fc9")) // this is an internal node of tr0
	if err != nil {
		t.Fatal(err)
	}
	err = tr1.Insert([]byte{0x04}, []byte{0x04})
	if err != nil {
		t.Fatal(err)
	}

	if tr1.RootHash() == tr0.RootHash() {
		t.Errorf("tr0 and tr1 have matching root hashes: %s", tr1.RootHash())
	}
}

func TestRootHashInsertQuickCheck(t *testing.T) {
	tr := new(Tree)

	keys := [][]byte{}
	f := func(b [32]byte) bool {
		before := tr.RootHash()
		h := sha3.Sum256(b[:])
		err := tr.Insert(b[:], h[:])
		keys = append(keys, b[:])
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
	_, hashes := makeVals(5)
	tr := &Tree{
		root: &node{key: bools("11111111"), hash: &hashes[0], isLeaf: true},
	}
	got := tr.lookup(tr.root, bitKey(bits("11111111")))
	if !reflect.DeepEqual(got, tr.root) {
		t.Log("lookup on 1-node tree")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root, 0))
	}

	tr = &Tree{
		root: &node{key: bools("11111110"), hash: &hashes[1], isLeaf: true},
	}
	got = tr.lookup(tr.root, bitKey(bits("11111111")))
	if got != nil {
		t.Log("lookup nonexistent key on 1-node tree")
		t.Fatalf("got:\n%swant nil", prettyNode(got, 0))
	}

	tr = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[2], hashes[1])),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[2], isLeaf: true},
				{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
			},
		},
	}
	got = tr.lookup(tr.root, bitKey(bits("11110000")))
	if !reflect.DeepEqual(got, tr.root.children[0]) {
		t.Log("lookup root's first child")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[0], 0))
	}

	tr = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[2], hashForNonLeaf(hashes[3], hashes[1]))),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[2], isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashes[3], hashes[1])),
					children: [2]*node{
						{key: bools("11111100"), hash: &hashes[3], isLeaf: true},
						{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
					},
				},
			},
		},
	}
	got = tr.lookup(tr.root, bitKey(bits("11111100")))
	if !reflect.DeepEqual(got, tr.root.children[1].children[0]) {
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[1].children[0], 0))
	}
}

func TestContains(t *testing.T) {
	vals, hashes := makeVals(4)
	tr := &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[2], hashForNonLeaf(hashes[3], hashes[1]))),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[2], isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashes[3], hashes[1])),
					children: [2]*node{
						{key: bools("11111100"), hash: &hashes[3], isLeaf: true},
						{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
					},
				},
			},
		},
	}
	contains := tr.Contains(bits("11111100"), vals[3])
	if !contains {
		t.Errorf("expected tree to contain %v, %x, but did not", bits("11111100"), vals[3])
	}

	contains = tr.Contains(bits("11111111"), vals[3])
	if contains {
		t.Errorf("expected tree to not contain %v, %x, but did", bits("11111111"), vals[3])
	}
}

func TestInsert(t *testing.T) {
	tr := new(Tree)
	vals, hashes := makeVals(6)

	tr.Insert(bits("11111111"), vals[0])
	tr.RootHash()
	want := &Tree{
		root: &node{key: bools("11111111"), hash: &hashes[0], isLeaf: true},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		log.Printf("want hash? %s", hashes[0])
		t.Log("insert into empty tree")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111111"), vals[1])
	tr.RootHash()
	want = &Tree{
		root: &node{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Log("inserting the same key updates the value, does not add a new node")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11110000"), vals[2])
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[2], hashes[1])),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[2], isLeaf: true},
				{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Log("different key creates a fork")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111100"), vals[3])
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[2], hashForNonLeaf(hashes[3], hashes[1]))),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[2], isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashes[3], hashes[1])),
					children: [2]*node{
						{key: bools("11111100"), hash: &hashes[3], isLeaf: true},
						{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111110"), vals[4])
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[2], hashForNonLeaf(hashes[3], hashForNonLeaf(hashes[4], hashes[1])))),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[2], isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashes[3], hashForNonLeaf(hashes[4], hashes[1]))),
					children: [2]*node{
						{key: bools("11111100"), hash: &hashes[3], isLeaf: true},
						{
							key:  bools("1111111"),
							hash: hashPtr(hashForNonLeaf(hashes[4], hashes[1])),
							children: [2]*node{
								{key: bools("11111110"), hash: &hashes[4], isLeaf: true},
								{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
							},
						},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Log("a fork is created for each level of similar key")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111011"), vals[5])
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[2], hashForNonLeaf(hashes[5], hashForNonLeaf(hashes[3], hashForNonLeaf(hashes[4], hashes[1]))))),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[2], isLeaf: true},
				{
					key:  bools("11111"),
					hash: hashPtr(hashForNonLeaf(hashes[5], hashForNonLeaf(hashes[3], hashForNonLeaf(hashes[4], hashes[1])))),
					children: [2]*node{
						{key: bools("11111011"), hash: &hashes[5], isLeaf: true},
						{
							key:  bools("111111"),
							hash: hashPtr(hashForNonLeaf(hashes[3], hashForNonLeaf(hashes[4], hashes[1]))),
							children: [2]*node{
								{key: bools("11111100"), hash: &hashes[3], isLeaf: true},
								{
									key:  bools("1111111"),
									hash: hashPtr(hashForNonLeaf(hashes[4], hashes[1])),
									children: [2]*node{
										{key: bools("11111110"), hash: &hashes[4], isLeaf: true},
										{key: bools("11111111"), hash: &hashes[1], isLeaf: true},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Log("compressed branch node is split")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}
}

func TestDelete(t *testing.T) {
	tr := new(Tree)
	_, hashes := makeVals(4)
	tr.root = &node{
		key:  bools("1111"),
		hash: hashPtr(hashForNonLeaf(hashes[0], hashForNonLeaf(hashes[1], hashForNonLeaf(hashes[2], hashes[3])))),
		children: [2]*node{
			{key: bools("11110000"), hash: &hashes[0], isLeaf: true},
			{
				key:  bools("111111"),
				hash: hashPtr(hashForNonLeaf(hashes[1], hashForNonLeaf(hashes[2], hashes[3]))),
				children: [2]*node{
					{key: bools("11111100"), hash: &hashes[1], isLeaf: true},
					{
						key:  bools("1111111"),
						hash: hashPtr(hashForNonLeaf(hashes[2], hashes[3])),
						children: [2]*node{
							{key: bools("11111110"), hash: &hashes[2], isLeaf: true},
							{key: bools("11111111"), hash: &hashes[3], isLeaf: true},
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
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[0], hashForNonLeaf(hashes[1], hashes[3]))),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[0], isLeaf: true},
				{
					key:  bools("111111"),
					hash: hashPtr(hashForNonLeaf(hashes[1], hashes[3])),
					children: [2]*node{
						{key: bools("11111100"), hash: &hashes[1], isLeaf: true},
						{key: bools("11111111"), hash: &hashes[3], isLeaf: true},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111100"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:  bools("1111"),
			hash: hashPtr(hashForNonLeaf(hashes[0], hashes[3])),
			children: [2]*node{
				{key: bools("11110000"), hash: &hashes[0], isLeaf: true},
				{key: bools("11111111"), hash: &hashes[3], isLeaf: true},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110011"))
	tr.RootHash()
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110000"))
	tr.RootHash()
	want = &Tree{
		root: &node{key: bools("11111111"), hash: &hashes[3], isLeaf: true},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111111"))
	tr.RootHash()
	want = &Tree{}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
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

		if !reflect.DeepEqual(g, c.w) {
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

		if !reflect.DeepEqual(g, c.w) {
			t.Errorf("byteKey(%#v) = %x want %x", c.b, g, c.w)
		}
	}
}

func makeVals(num int) (vals [][]byte, hashes []bc.Hash) {
	for i := 0; i < num; i++ {
		v := sha3.Sum256([]byte{byte(i)})
		vals = append(vals, v[:])
		hashes = append(hashes, sha3.Sum256(append([]byte{0x00}, v[:]...)))
	}
	return vals, hashes
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

func hashForNonLeaf(a, b bc.Hash) bc.Hash {
	d := []byte{0x01}
	d = append(d, a[:]...)
	d = append(d, b[:]...)
	return sha3.Sum256(d)
}

func hashPtr(h bc.Hash) *bc.Hash {
	return &h
}

func mustDecodeHash(s string) []byte {
	var h bc.Hash
	err := h.UnmarshalText([]byte(strings.TrimSpace(s)))
	if err != nil {
		log.Fatalf("error decoding hash: %s", err)
	}
	b := [32]byte(h)
	return b[:]
}
