package patricia

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
)

func TestRootHashBug(t *testing.T) {
	tr := NewTree(nil)

	err := tr.Insert([]byte{0x94}, HashValuer(bc.Hash{0x01}))
	if err != nil {
		t.Fatal(err)
	}
	err = tr.Insert([]byte{0x36}, HashValuer(bc.Hash{0x02}))
	if err != nil {
		t.Fatal(err)
	}
	before := tr.RootHash()
	err = tr.Insert([]byte{0xba}, HashValuer(bc.Hash{0x03}))
	if err != nil {
		t.Fatal(err)
	}
	if tr.RootHash() == before {
		t.Errorf("before and after root hash is %s", before)
	}
}

func TestRootHashInsertQuickCheck(t *testing.T) {
	tr := NewTree(nil)

	keys := [][]byte{}
	f := func(b [32]byte) bool {
		before := tr.RootHash()
		err := tr.Insert(b[:], HashValuer(b))
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
	vals := makeVals(5)
	tr := &Tree{
		root: &Node{key: bools("11111111"), val: vals[0], isLeaf: true},
	}
	got := tr.Lookup(bits("11111111"))
	if !reflect.DeepEqual(got, tr.root) {
		t.Log("lookup on 1-node tree")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root, 0))
	}

	tr = &Tree{
		root: &Node{key: bools("11111110"), val: vals[1], isLeaf: true},
	}
	got = tr.Lookup(bits("11111111"))
	if got != nil {
		t.Log("lookup nonexistent key on 1-node tree")
		t.Fatalf("got:\n%swant nil", prettyNode(got, 0))
	}

	tr = &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[2], isLeaf: true},
				{key: bools("11111111"), val: vals[1], isLeaf: true},
			},
		},
	}
	got = tr.Lookup(bits("11110000"))
	if !reflect.DeepEqual(got, tr.root.children[0]) {
		t.Log("lookup root's first child")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[0], 0))
	}

	tr = &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[2], isLeaf: true},
				{
					key: bools("111111"),
					children: [2]*Node{
						{key: bools("11111100"), val: vals[3], isLeaf: true},
						{key: bools("11111111"), val: vals[1], isLeaf: true},
					},
				},
			},
		},
	}
	got = tr.Lookup(bits("11111100"))
	if !reflect.DeepEqual(got, tr.root.children[1].children[0]) {
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[1].children[0], 0))
	}
}

func TestInsert(t *testing.T) {
	tr := NewTree(nil)
	vals := makeVals(6)
	tr.Insert(bits("11111111"), vals[0])

	want := &Tree{
		root: &Node{key: bools("11111111"), val: vals[0], isLeaf: true},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Log("insert into empty tree")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111111"), vals[1])
	want = &Tree{
		root: &Node{key: bools("11111111"), val: vals[1], isLeaf: true},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Log("inserting the same key updates the value, does not add a new node")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11110000"), vals[2])
	want = &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[2], isLeaf: true},
				{key: bools("11111111"), val: vals[1], isLeaf: true},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Log("different key creates a fork")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111100"), vals[3])
	want = &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[2], isLeaf: true},
				{
					key: bools("111111"),
					children: [2]*Node{
						{key: bools("11111100"), val: vals[3], isLeaf: true},
						{key: bools("11111111"), val: vals[1], isLeaf: true},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111110"), vals[4])
	want = &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[2], isLeaf: true},
				{
					key: bools("111111"),
					children: [2]*Node{
						{key: bools("11111100"), val: vals[3], isLeaf: true},
						{
							key: bools("1111111"),
							children: [2]*Node{
								{key: bools("11111110"), val: vals[4], isLeaf: true},
								{key: bools("11111111"), val: vals[1], isLeaf: true},
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
	want = &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[2], isLeaf: true},
				{
					key: bools("11111"),
					children: [2]*Node{
						{key: bools("11111011"), val: vals[5], isLeaf: true},
						{
							key: bools("111111"),
							children: [2]*Node{
								{key: bools("11111100"), val: vals[3], isLeaf: true},
								{
									key: bools("1111111"),
									children: [2]*Node{
										{key: bools("11111110"), val: vals[4], isLeaf: true},
										{key: bools("11111111"), val: vals[1], isLeaf: true},
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
	vals := makeVals(4)
	tr := NewTree(nil)
	tr.root = &Node{
		key: bools("1111"),
		children: [2]*Node{
			{key: bools("11110000"), val: vals[0], isLeaf: true},
			{
				key: bools("111111"),
				children: [2]*Node{
					{key: bools("11111100"), val: vals[1], isLeaf: true},
					{
						key: bools("1111111"),
						children: [2]*Node{
							{key: bools("11111110"), val: vals[2], isLeaf: true},
							{key: bools("11111111"), val: vals[3], isLeaf: true},
						},
					},
				},
			},
		},
	}

	tr.Delete(bits("11111110"))
	want := &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[0], isLeaf: true},
				{
					key: bools("111111"),
					children: [2]*Node{
						{key: bools("11111100"), val: vals[1], isLeaf: true},
						{key: bools("11111111"), val: vals[3], isLeaf: true},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111100"))
	want = &Tree{
		root: &Node{
			key: bools("1111"),
			children: [2]*Node{
				{key: bools("11110000"), val: vals[0], isLeaf: true},
				{key: bools("11111111"), val: vals[3], isLeaf: true},
			},
		},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110011"))
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110000"))
	want = &Tree{
		root: &Node{key: bools("11111111"), val: vals[3], isLeaf: true},
	}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111111"))
	want = &Tree{}
	if !reflect.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}
}

func TestRootHash(t *testing.T) {
	vals := makeVals(3)
	cases := []struct {
		tree *Tree
		want bc.Hash
	}{{
		tree: &Tree{},
		want: bc.Hash{},
	}, {
		tree: &Tree{root: &Node{val: vals[0], isLeaf: true}},
		want: vals[0].Value().Hash(),
	}, {
		tree: &Tree{
			root: &Node{
				children: [2]*Node{{val: vals[0], isLeaf: true}, {val: vals[1], isLeaf: true}},
			},
		},
		want: hash(vals[0].Value().Hash(), vals[1].Value().Hash()),
	}, {
		tree: &Tree{
			root: &Node{
				children: [2]*Node{
					{
						children: [2]*Node{{val: vals[0], isLeaf: true}, {val: vals[1], isLeaf: true}},
					},
					{val: vals[2], isLeaf: true},
				},
			},
		},
		want: hash(hash(vals[0].Value().Hash(), vals[1].Value().Hash()), vals[2].Value().Hash()),
	}}
	for _, c := range cases {
		got := c.tree.RootHash()

		if !bytes.Equal(got[:], c.want[:]) {
			t.Errorf("got %s want %s", got, c.want)
		}
	}
}

func TestBoolKey(t *testing.T) {
	cases := []struct {
		b []byte
		w []uint8
	}{{
		b: nil,
		w: nil,
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

func makeVals(num int) []Valuer {
	var vals []Valuer
	for i := 0; i < num; i++ {
		vals = append(vals, BytesValuer([]byte{byte(i)}))
	}
	return vals
}

func pretty(t *Tree) string {
	if t.root == nil {
		return ""
	}
	return prettyNode(t.root, 0)
}

func prettyNode(n *Node, depth int) string {
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
	if n.isLeaf {
		prettyStr += fmt.Sprintf(" val=%+v", n.val)
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

func hash(a, b bc.Hash) bc.Hash {
	var d []byte
	d = append(d, a[:]...)
	d = append(d, b[:]...)
	return sha3.Sum256(d)
}
