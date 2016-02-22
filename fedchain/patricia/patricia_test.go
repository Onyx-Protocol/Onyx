package patricia

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"chain/crypto/hash256"
	"chain/fedchain/bc"
)

func TestInsert(t *testing.T) {
	tr := &Tree{}
	vals := makeVals(6)
	tr.Insert(bits("11111111"), vals[0])

	want := &Tree{
		root: &node{key: bools("11111111"), val: vals[0]},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Log("insert into empty tree")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111111"), vals[1])
	want = &Tree{
		root: &node{key: bools("11111111"), val: vals[1]},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Log("inserting the same key updates the value, does not add a new node")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11110000"), vals[2])
	want = &Tree{
		root: &node{
			key: bools("1111"),
			children: [2]*node{
				{key: []uint8{0, 0, 0, 0}, val: vals[2]},
				{key: []uint8{1, 1, 1, 1}, val: vals[1]},
			},
		},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Log("different key creates a fork")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111100"), vals[3])
	want = &Tree{
		root: &node{
			key: bools("1111"),
			children: [2]*node{
				{key: []uint8{0, 0, 0, 0}, val: vals[2]},
				{
					key: []uint8{1, 1},
					children: [2]*node{
						{key: []uint8{0, 0}, val: vals[3]},
						{key: []uint8{1, 1}, val: vals[1]},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111110"), vals[4])
	want = &Tree{
		root: &node{
			key: bools("1111"),
			children: [2]*node{
				{key: []uint8{0, 0, 0, 0}, val: vals[2]},
				{
					key: []uint8{1, 1},
					children: [2]*node{
						{key: []uint8{0, 0}, val: vals[3]},
						{
							key: []uint8{1},
							children: [2]*node{
								{key: []uint8{0}, val: vals[4]},
								{key: []uint8{1}, val: vals[1]},
							},
						},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Log("a fork is created for each level of similar key")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111011"), vals[5])
	want = &Tree{
		root: &node{
			key: bools("1111"),
			children: [2]*node{
				{key: []uint8{0, 0, 0, 0}, val: vals[2]},
				{
					key: []uint8{1},
					children: [2]*node{
						{key: []uint8{0, 1, 1}, val: vals[5]},
						{
							key: []uint8{1},
							children: [2]*node{
								{key: []uint8{0, 0}, val: vals[3]},
								{
									key: []uint8{1},
									children: [2]*node{
										{key: []uint8{0}, val: vals[4]},
										{key: []uint8{1}, val: vals[1]},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Log("compressed branch node is split")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}
}

func TestDelete(t *testing.T) {
	vals := makeVals(4)
	tr := &Tree{
		root: &node{
			key: bools("1111"),
			children: [2]*node{
				{key: []uint8{0, 0, 0, 0}, val: vals[1]},
				{
					key: []uint8{1, 1},
					children: [2]*node{
						{key: []uint8{0, 0}, val: vals[2]},
						{
							key: []uint8{1},
							children: [2]*node{
								{key: []uint8{0}, val: vals[3]},
								{key: []uint8{1}, val: vals[0]},
							},
						},
					},
				},
			},
		},
	}

	tr.Delete(bits("11111110"))
	want := &Tree{
		root: &node{
			key: bools("1111"),
			children: [2]*node{
				{key: []uint8{0, 0, 0, 0}, val: vals[1]},
				{
					key: []uint8{1, 1},
					children: [2]*node{
						{key: []uint8{0, 0}, val: vals[2]},
						{key: []uint8{1, 1}, val: vals[0]},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111100"))
	want = &Tree{
		root: &node{
			key: bools("1111"),
			children: [2]*node{
				{key: []uint8{0, 0, 0, 0}, val: vals[1]},
				{key: []uint8{1, 1, 1, 1}, val: vals[0]},
			},
		},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110011"))
	if !reflect.DeepEqual(tr, want) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110000"))
	want = &Tree{
		root: &node{key: bools("11111111"), val: vals[0]},
	}
	if !reflect.DeepEqual(tr, want) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111111"))
	want = &Tree{root: nil}
	if !reflect.DeepEqual(tr, want) {
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
		tree: &Tree{root: &node{val: vals[0]}},
		want: vals[0].Hash(),
	}, {
		tree: &Tree{root: &node{
			children: [2]*node{{val: vals[0]}, {val: vals[1]}},
		}},
		want: hash(vals[0].Hash(), vals[1].Hash()),
	}, {
		tree: &Tree{root: &node{
			children: [2]*node{
				{
					children: [2]*node{{val: vals[0]}, {val: vals[1]}},
				},
				{val: vals[2]},
			},
		}},
		want: hash(hash(vals[0].Hash(), vals[1].Hash()), vals[2].Hash()),
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
		g := boolKey(c.b)

		if !reflect.DeepEqual(g, c.w) {
			t.Errorf("Key(0x%x) = %v want %v", c.b, g, c.w)
		}
	}
}

func TestFork(t *testing.T) {
	vals := makeVals(2)
	cases := []struct {
		n     *node
		after int
		want  *node
	}{{
		n:     &node{key: []uint8{0, 0, 1, 1}, val: vals[0]},
		after: 2,
		want: &node{
			key:      []uint8{0, 0},
			children: [2]*node{nil, {key: []uint8{1, 1}, val: vals[0]}},
		},
	}, {
		n:     &node{key: []uint8{1, 1, 0, 1}, val: vals[0]},
		after: 2,
		want: &node{
			key:      []uint8{1, 1},
			children: [2]*node{{key: []uint8{0, 1}, val: vals[0]}, nil},
		},
	}, {
		n: &node{
			key: []uint8{1, 1, 1, 1},
			children: [2]*node{
				{key: []uint8{0}, val: vals[0]},
				{key: []uint8{1}, val: vals[1]},
			},
		},
		after: 2,
		want: &node{
			key: []uint8{1, 1},
			children: [2]*node{
				nil,
				&node{
					key: []uint8{1, 1},
					children: [2]*node{
						{key: []uint8{0}, val: vals[0]},
						{key: []uint8{1}, val: vals[1]},
					},
				},
			},
		},
	}}

	for _, c := range cases {
		c.n.fork(c.after)
		if !reflect.DeepEqual(c.n, c.want) {
			t.Errorf("got:\n%swant:\n%s", prettyNode(c.n, 0), prettyNode(c.want, 0))
		}
	}
}

func makeVals(num int) []Hasher {
	var vals []Hasher
	for i := 0; i < num; i++ {
		vals = append(vals, &bc.TxData{Metadata: []byte{uint8(i)}})
	}
	return vals
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
	if n.isLeaf() {
		prettyStr += fmt.Sprintf(" val=%+v", n.val)
	}
	prettyStr += "\n"

	for _, c := range n.children {
		prettyStr += prettyNode(c, depth+1)
	}

	return prettyStr
}

func bits(lit string) []byte {
	var b [31]byte
	n, _ := strconv.ParseUint(lit, 2, 8)
	return append(b[:], byte(n))
}

func bools(lit string) []uint8 {
	b := boolKey(bits(lit))
	return append(b[:31*8], b[31*8+8-len(lit):]...)
}

func hash(a, b bc.Hash) bc.Hash {
	var d []byte
	d = append(d, a[:]...)
	d = append(d, b[:]...)
	return hash256.Sum(d)
}
