package patricia

import (
	"bytes"

	"chain/cos/bc"
	"chain/crypto/hash256"
	"chain/errors"
)

// ErrPrefix is returned from Insert or Delete if
// the key provided is a prefix to existing nodes.
var ErrPrefix = errors.New("key provided is a prefix to other keys")

// Hasher is the interface used for values inserted
// into the tree. Since this tree is used to create
// a merkle root, each item must be hashable.
type Hasher interface {
	Hash() bc.Hash
}

// Tree is a patricia tree implementation, or a radix tree
// with a radix of 2 -- creating an uneven binary tree.
// Each entry is a key value pair. The key determines
// where the value is placed in the tree, with each bit
// of the key indicating a path.
//
// The nodes in the tree form an immutable persistent
// data structure, therefore Copy is a O(1) operation.
type Tree struct {
	root    *Node
	deletes map[string]bool
	updates map[string]*Node
	inserts map[string]*Node
}

// NewTree assembles a tree using a slice of nodes.
// The slice of nodes passed must form a complete
// breadth-first traversal of the entire tree, including
// interior nodes.
func NewTree(nodes []*Node) *Tree {
	tree := &Tree{
		deletes: make(map[string]bool),
		updates: make(map[string]*Node),
		inserts: make(map[string]*Node),
	}

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

// Delta returns all of the deletes, inserts, and updates
// that have occurred on the tree.
func (t *Tree) Delta() (deletes [][]uint8, inserts, updates []*Node) {
	for k := range t.deletes {
		deletes = append(deletes, []uint8(k))
	}
	for _, node := range t.inserts {
		inserts = append(inserts, node)
	}
	for _, node := range t.updates {
		updates = append(updates, node)
	}

	return deletes, inserts, updates
}

func (t *Tree) trackDelete(key []uint8) {
	if t.inserts[string(key)] != nil {
		delete(t.inserts, string(key))
		return
	}
	delete(t.updates, string(key))
	t.deletes[string(key)] = true
}

func (t *Tree) trackUpdate(n *Node) {
	if t.inserts[string(n.key)] != nil {
		t.inserts[string(n.key)] = n
		return
	}
	t.updates[string(n.key)] = n
}

func (t *Tree) trackInsert(n *Node) {
	if t.deletes[string(n.key)] {
		delete(t.deletes, string(n.key))
		t.updates[string(n.key)] = n
		return
	}
	t.inserts[string(n.key)] = n
}

// Insert enters data into the tree.
// If the key is not already present in the tree,
// a new node will be created and inserted,
// rearranging the tree to the optimal structure.
// If the key is present, the existing node is found
// and its value is updated, leaving the structure of
// the tree alone.
func (t *Tree) Insert(bkey []byte, val Hasher) error {
	key := bitKey(bkey)

	if t.root == nil {
		t.root = &Node{key: key, val: val, isLeaf: true}
		t.trackInsert(t.root)
		return nil
	}

	var err error
	t.root, err = t.insert(t.root, key, val)

	return err
}

func (t *Tree) insert(n *Node, key []uint8, val Hasher) (*Node, error) {
	if bytes.Equal(n.key, key) {
		if !n.isLeaf {
			return n, errors.Wrap(ErrPrefix)
		}
		n = &Node{
			isLeaf: true,
			key:    n.key,
			val:    val,
		}
		t.trackUpdate(n)
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
		t.trackUpdate(newNode)
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
	t.trackInsert(newNode)
	t.trackInsert(newNode.children[key[common]])
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
		t.trackDelete(key)
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
		t.trackDelete(n.key)
		return n.children[1-bit], nil
	}

	newNode := new(Node)
	*newNode = *n
	newNode.children[bit] = newChild

	t.trackUpdate(newNode)
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
	val      Hasher
}

// NewNode returns a node with the given key and hash
func NewNode(key []uint8, hash bc.Hash, isLeaf bool) *Node {
	return &Node{key: key, hash: &hash, isLeaf: isLeaf}
}

// Key returns the key for the current node
func (n *Node) Key() []uint8 { return n.key }

// IsLeaf returns whether the current node is a leaf node
func (n *Node) IsLeaf() bool { return n.isLeaf }

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
		return n.val.Hash()
	}

	var data []byte
	for _, c := range n.children {
		h := c.Hash()
		data = append(data, h[:]...)
	}

	return hash256.Sum(data)
}
