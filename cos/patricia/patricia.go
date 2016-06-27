package patricia

import (
	"bytes"

	"golang.org/x/crypto/sha3"

	"chain/cos/bc"
	"chain/errors"
)

// ErrPrefix is returned from Insert or Delete if
// the key provided is a prefix to existing nodes.
var ErrPrefix = errors.New("key provided is a prefix to other keys")

// Tree is a patricia tree implementation, or a radix tree
// with a radix of 2 -- creating an uneven binary tree.
// Each entry is a key value pair. The key determines
// where the value is placed in the tree, with each bit
// of the key indicating a path.
//
// The nodes in the tree form an immutable persistent
// data structure, therefore Copy is a O(1) operation.
type Tree struct {
	root *Node
}

// NewTree assembles a tree using a slice of nodes.
// The slice of nodes passed must form a complete
// breadth-first traversal of the entire tree, including
// interior nodes.
func NewTree(nodes []*Node) *Tree {
	tree := &Tree{}

	for _, node := range nodes {
		if tree.root == nil {
			tree.root = node
			continue
		}
		parent := tree.root
		for {
			next := parent.children[node.key[len(parent.key)]]
			if next == nil {
				parent.children[node.key[len(parent.key)]] = node
				break
			}
			parent = next
		}
	}

	return tree
}

// Copy returns a new tree with the same root as this tree
func Copy(t *Tree) *Tree {
	newT := NewTree(nil)
	newT.root = t.root
	return newT
}

// WalkFunc is the type of the function called for each node visited by
// Walk. If an error is returned, processing stops.
type WalkFunc func(n *Node) error

// Walk walks the patricia tree calling walkFn for each node in the tree,
// including root, in a pre-order traversal.
func Walk(t *Tree, walkFn WalkFunc) error {
	if t.root == nil {
		return nil
	}
	return walk(t.root, walkFn)
}

func walk(n *Node, walkFn WalkFunc) error {
	err := walkFn(n)
	if err != nil {
		return err
	}

	if !n.isLeaf {
		err = walk(n.children[0], walkFn)
		if err != nil {
			return err
		}

		err = walk(n.children[1], walkFn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Lookup looks up a key and returns a Node with the provided key.
// If the key is not present in the tree, nil is returned.
func (t *Tree) Lookup(bkey []byte) *Node {
	if t.root == nil {
		return nil
	}

	key := bitKey(bkey)
	return t.lookup(t.root, key)
}

func (t *Tree) lookup(n *Node, key []uint8) *Node {
	if bytes.Equal(n.key, key) {
		if !n.isLeaf {
			return nil
		}
		return n
	}
	if !bytes.HasPrefix(key, n.key) {
		return nil
	}

	bit := key[len(n.key)]
	return t.lookup(n.children[bit], key)
}

// Insert enters data into the tree.
// If the key is not already present in the tree,
// a new node will be created and inserted,
// rearranging the tree to the optimal structure.
// If the key is present, the existing node is found
// and its value is updated, leaving the structure of
// the tree alone.
func (t *Tree) Insert(bkey []byte, val Valuer) error {
	key := bitKey(bkey)

	if t.root == nil {
		t.root = &Node{key: key, val: val, isLeaf: true}
		return nil
	}

	var err error
	t.root, err = t.insert(t.root, key, val)

	return err
}

func (t *Tree) insert(n *Node, key []uint8, val Valuer) (*Node, error) {
	if bytes.Equal(n.key, key) {
		if !n.isLeaf {
			return n, errors.Wrap(ErrPrefix)
		}
		n = &Node{
			isLeaf: true,
			key:    n.key,
			val:    val,
		}
		return n, nil
	}

	if bytes.HasPrefix(key, n.key) {
		if n.isLeaf {
			return n, errors.Wrap(ErrPrefix)
		}
		bit := key[len(n.key)]

		child := n.children[bit]
		child, err := t.insert(child, key, val)
		if err != nil {
			return n, err
		}
		newNode := new(Node)
		*newNode = *n
		newNode.children[bit] = child // mutation is ok because newNode hasn't escaped yet
		return newNode, nil
	}

	common := len(commonPrefix(n.key, key))
	newNode := &Node{
		key: n.key[:common],
	}
	newNode.children[key[common]] = &Node{
		key:    key,
		val:    val,
		isLeaf: true,
	}
	newNode.children[1-key[common]] = n
	return newNode, nil
}

// Delete removes up to one value with a matching key.
// After removing the node, it will rearrange the tree
// to the optimal structure.
func (t *Tree) Delete(bkey []byte) error {
	key := bitKey(bkey)

	if t.root == nil {
		return nil
	}

	var err error
	t.root, err = t.delete(t.root, key)
	return err
}

func (t *Tree) delete(n *Node, key []uint8) (*Node, error) {
	if bytes.Equal(key, n.key) {
		if !n.isLeaf {
			return n, errors.Wrap(ErrPrefix)
		}
		return nil, nil
	}

	if !bytes.HasPrefix(key, n.key) {
		return n, nil
	}

	bit := key[len(n.key)]
	newChild, err := t.delete(n.children[bit], key)
	if err != nil {
		return nil, err
	}

	if newChild == nil {
		return n.children[1-bit], nil
	}

	newNode := new(Node)
	*newNode = *n
	newNode.children[bit] = newChild

	return newNode, nil
}

// RootHash returns the merkle root of the tree
func (t *Tree) RootHash() bc.Hash {
	root := t.root
	if root == nil {
		return bc.Hash{}
	}
	return root.Hash()
}

// bitKey takes a byte array and returns a key that can
// be used inside insert and delete operations.
func bitKey(byteKey []byte) []uint8 {
	var key []uint8
	for _, b := range byteKey {
		for i := uint(0); i < 8; i++ {
			key = append(key, (b>>(7-i))&1)
		}
	}
	return key
}

func commonPrefix(a, b []uint8) []uint8 {
	var common []uint8
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
		common = append(common, a[i])
	}
	return common
}

// Node is a leaf or branch node in a tree
type Node struct {
	key      []uint8
	hash     *bc.Hash
	isLeaf   bool
	children [2]*Node
	val      Valuer
}

// Value encapsulates a value stored in the tree. The value may
// be a precomputed hash or an abritrary byte slice that will be
// hashed when necessary.
type Value struct {
	Bytes  []byte
	IsHash bool
}

// Hash returns a hash of the value.
func (v Value) Hash() bc.Hash {
	if !v.IsHash {
		return sha3.Sum256(v.Bytes)
	}

	var h bc.Hash
	copy(h[:], v.Bytes)
	return h
}

// NewNode returns a node with the given key and hash
func NewNode(key []uint8, v Valuer, isLeaf bool) *Node {
	return &Node{key: key, val: v, isLeaf: isLeaf}
}

// Key returns the key for the current node
func (n *Node) Key() []uint8 { return n.key }

// IsLeaf returns whether the current node is a leaf node
func (n *Node) IsLeaf() bool { return n.isLeaf }

// Value returns the raw Value for this node.
func (n *Node) Value() Value {
	if n.val == nil {
		return Value{}
	}
	return n.val.Value()
}

// Hash will return a cached hash if available,
// if not it will calculate a new hash and cache it.
func (n *Node) Hash() bc.Hash {
	if n.hash != nil {
		return *n.hash
	}

	hash := n.calcHash()
	n.hash = &hash

	return hash
}

func (n *Node) calcHash() bc.Hash {
	if n.isLeaf {
		return n.val.Value().Hash()
	}

	var data []byte
	for _, c := range n.children {
		h := c.Hash()
		data = append(data, h[:]...)
	}

	return sha3.Sum256(data)
}

// Valuer describes types that can produce a patricia tree value.
type Valuer interface {
	Value() Value
}

// HashValuer returns a Valuer representing a precomputed hash.
func HashValuer(h bc.Hash) Valuer {
	return literalValuer(Value{Bytes: h[:], IsHash: true})
}

// BytesValuer returns a Valuer representing the provided bytes.
func BytesValuer(b []byte) Valuer {
	return literalValuer(Value{Bytes: b, IsHash: false})
}

type literalValuer Value

func (v literalValuer) Value() Value { return Value(v) }
