package patricia

import (
	"bytes"

	"chain/crypto/hash256"
	"chain/errors"
	"chain/fedchain/bc"
)

const keyLen = 256

// ErrKeyLength is returned if a key passed into
// Insert or Delete is not the required length.
var ErrKeyLength = errors.New("key must be 256 bits long")

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
// The zero value of Tree is an empty tree, ready to use.
type Tree struct {
	root *node
}

// Insert enters data into the tree.
// If the key is not already present in the tree,
// a new node will be created and inserted,
// rearranging the tree to the optimal structure.
// If the key is present, the existing node is found
// and its value is updated, leaving the structure of
// the tree alone.
func (t *Tree) Insert(bkey []byte, val Hasher) error {
	key := boolKey(bkey)

	if len(key) != keyLen {
		return ErrKeyLength
	}

	if t.root == nil {
		t.root = &node{key: key, val: val}
		return nil
	}

	insert(t.root, key, val)
	return nil
}

func insert(n *node, key []uint8, val Hasher) {
	if bytes.Equal(n.key, key) {
		n.val = val
		return
	}

	if bytes.Equal(n.key, key[:len(n.key)]) {
		child := n.children[key[len(n.key)]]
		insert(child, key[len(n.key):], val)
		return
	}

	common := len(commonPrefix(n.key, key))
	n.fork(common)
	n.children[key[common]] = &node{key: key[common:], val: val}
}

// Delete removes up to one value with a matching key.
// After removing the node, it will rearrange the tree
// to the optimal structure.
func (t *Tree) Delete(bkey []byte) error {
	key := boolKey(bkey)
	if len(key) != keyLen {
		return ErrKeyLength
	}

	if t.root == nil {
		return nil
	}

	if bytes.Equal(t.root.key, key) {
		t.root = nil
		return nil
	}

	delete(t.root, key)

	return nil
}

func delete(n *node, key []uint8) {
	if !bytes.Equal(n.key, key[:len(n.key)]) {
		return
	}

	key = key[len(n.key):]
	child := n.children[key[0]]
	if bytes.Equal(child.key, key) {
		n.children[key[0]] = nil
		n.merge(n.children[1-key[0]])
		return
	}

	delete(child, key)
}

// RootHash returns the merkle root of the tree
func (t *Tree) RootHash() bc.Hash {
	if t.root == nil {
		return bc.Hash{}
	}
	return t.root.hash()
}

// boolKey takes a byte array and returns a key that can
// be used inside insert and delete operations.
func boolKey(byteKey []byte) []uint8 {
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

type node struct {
	key      []uint8
	children [2]*node
	val      Hasher
}

func (n *node) isLeaf() bool {
	return n.val != nil
}

func (n *node) hash() bc.Hash {
	if n.isLeaf() {
		return n.val.Hash()
	}

	var data []byte
	for _, c := range n.children {
		h := c.hash()
		data = append(data, h[:]...)
	}

	return hash256.Sum(data)
}

func (n *node) merge(child *node) {
	n.key = append(n.key, child.key...)
	n.val = child.val
	n.children = child.children
}

func (n *node) fork(after int) {
	newLeaf := &node{key: n.key[after:], val: n.val, children: n.children}
	n.children = [2]*node{nil, nil}
	n.children[n.key[after]] = newLeaf
	n.key = n.key[:after]
	if len(n.key) == 0 {
		n.key = nil
	}
	n.val = nil
}
