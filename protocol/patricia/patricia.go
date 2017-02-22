// Package patricia implements a patricia tree, or a radix
// tree with a radix of 2 -- creating an uneven binary tree.
//
// Each entry is a key value pair. The key determines
// where the value is placed in the tree, with each bit
// of the key indicating a path. Values are arbitrary byte
// slices but only the SHA3-256 hash of the value is stored
// within the tree.
//
// The nodes in the tree form an immutable persistent data
// structure, therefore Copy is a O(1) operation.
package patricia

import (
	"bytes"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
)

var (
	leafPrefix     = []byte{0x00}
	interiorPrefix = []byte{0x01}
)

// Tree implements a patricia tree.
type Tree struct {
	root *node
}

// Copy returns a new tree with the same root as this tree. It
// is an O(1) operation.
func Copy(t *Tree) *Tree {
	newT := new(Tree)
	newT.root = t.root
	return newT
}

// WalkFunc is the type of the function called for each leaf
// visited by Walk. If an error is returned, processing stops.
type WalkFunc func(k []byte) error

// Walk walks the patricia tree calling walkFn for each leaf in
// the tree. If an error is returned by walkFn at any point,
// processing is stopped and the error is returned.
func Walk(t *Tree, walkFn WalkFunc) error {
	if t.root == nil {
		return nil
	}
	return walk(t.root, walkFn)
}

func walk(n *node, walkFn WalkFunc) error {
	if n.isLeaf {
		return walkFn(n.Key())
	}

	err := walk(n.children[0], walkFn)
	if err != nil {
		return err
	}

	err = walk(n.children[1], walkFn)
	return err
}

// ContainsKey returns true if the key contains the provided
// key, without checking its corresponding hash.
func (t *Tree) ContainsKey(bkey []byte) bool {
	if t.root == nil {
		return false
	}
	return lookup(t.root, bitKey(bkey)) != nil
}

// Contains returns true if the tree contains the provided
// key, value pair.
func (t *Tree) Contains(bkey, val []byte) bool {
	if t.root == nil {
		return false
	}

	key := bitKey(bkey)
	n := lookup(t.root, key)

	var hash bc.Hash
	h := sha3pool.Get256()
	h.Write(leafPrefix)
	h.Write(val[:])
	h.Read(hash[:])
	sha3pool.Put256(h)
	return n != nil && n.Hash() == hash
}

func lookup(n *node, key []uint8) *node {
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
	return lookup(n.children[bit], key)
}

// Insert enters data into the tree.
// If the key is not already present in the tree,
// a new node will be created and inserted,
// rearranging the tree to the optimal structure.
// If the key is present, the existing node is found
// and its value is updated, leaving the structure of
// the tree alone.
// It is an error for bkey to be a prefix
// of a key already in t or to contain a key already
// in t as a prefix.
func (t *Tree) Insert(bkey, val []byte) error {
	key := bitKey(bkey)

	var hash bc.Hash
	h := sha3pool.Get256()
	h.Write(leafPrefix)
	h.Write(val)
	h.Read(hash[:])
	sha3pool.Put256(h)

	if t.root == nil {
		t.root = &node{key: key, hash: &hash, isLeaf: true}
		return nil
	}

	var err error
	t.root, err = insert(t.root, key, &hash)
	return err
}

func insert(n *node, key []uint8, hash *bc.Hash) (*node, error) {
	if bytes.Equal(n.key, key) {
		if !n.isLeaf {
			return n, errors.Wrap(errors.New("key provided is a prefix to other keys"))
		}

		n = &node{
			isLeaf: true,
			key:    n.key,
			hash:   hash,
		}
		return n, nil
	}

	if bytes.HasPrefix(key, n.key) {
		if n.isLeaf {
			return n, errors.Wrap(errors.New("key provided is a prefix to other keys"))
		}
		bit := key[len(n.key)]

		child := n.children[bit]
		child, err := insert(child, key, hash)
		if err != nil {
			return n, err
		}
		newNode := new(node)
		*newNode = *n
		newNode.children[bit] = child // mutation is ok because newNode hasn't escaped yet
		newNode.hash = nil
		return newNode, nil
	}

	common := commonPrefixLen(n.key, key)
	newNode := &node{
		key: key[:common],
	}
	newNode.children[key[common]] = &node{
		key:    key,
		hash:   hash,
		isLeaf: true,
	}
	newNode.children[1-key[common]] = n
	return newNode, nil
}

// Delete removes up to one value with a matching key.
// After removing the node, it will rearrange the tree
// to the optimal structure.
func (t *Tree) Delete(bkey []byte) {
	key := bitKey(bkey)

	if t.root != nil {
		t.root = delete(t.root, key)
	}
}

func delete(n *node, key []uint8) *node {
	if bytes.Equal(key, n.key) {
		if !n.isLeaf {
			return n
		}
		return nil
	}

	if !bytes.HasPrefix(key, n.key) {
		return n
	}

	bit := key[len(n.key)]
	newChild := delete(n.children[bit], key)

	if newChild == nil {
		return n.children[1-bit]
	}

	newNode := new(node)
	*newNode = *n
	newNode.key = newChild.key[:len(n.key)] // only use slices of leaf node keys
	newNode.children[bit] = newChild
	newNode.hash = nil

	return newNode
}

// RootHash returns the merkle root of the tree.
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
	key := make([]uint8, 0, len(byteKey)*8)
	for _, b := range byteKey {
		for i := uint(0); i < 8; i++ {
			key = append(key, (b>>(7-i))&1)
		}
	}
	return key
}

// byteKey is the inverse of bitKey.
func byteKey(bitKey []uint8) (key []byte) {
	key = make([]byte, len(bitKey)/8)
	for i := uint(0); i < uint(len(key)); i++ {
		var b byte
		for j := uint(0); j < 8; j++ {
			bit := bitKey[i*8+j]
			b |= bit << (7 - j)
		}
		key[i] = b
	}
	return key
}

func commonPrefixLen(a, b []uint8) int {
	var common int
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
		common++
	}
	return common
}

// node is a leaf or branch node in a tree
type node struct {
	key      []uint8
	hash     *bc.Hash
	isLeaf   bool
	children [2]*node
}

// Key returns the key for the current node as bytes, as it
// was provided to Insert.
func (n *node) Key() []byte { return byteKey(n.key) }

// Hash will return the hash for this node.
func (n *node) Hash() bc.Hash {
	n.calcHash()
	return *n.hash
}

func (n *node) calcHash() {
	if n.hash != nil {
		return
	}

	h := sha3pool.Get256()
	h.Write(interiorPrefix)
	for _, c := range n.children {
		c.calcHash()
		h.Write(c.hash[:])
	}

	var hash bc.Hash
	h.Read(hash[:])
	n.hash = &hash
	sha3pool.Put256(h)
}
