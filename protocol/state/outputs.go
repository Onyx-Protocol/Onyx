package state

import "chain/protocol/bc"

// OutputTreeItem returns the key of an output in the state tree,
// as well as the output commitment (a second []byte) for Inserts
// into the state tree.
func OutputTreeItem(outputID bc.OutputID) (bkey, commitment []byte) {
	// We implement the set of unspent IDs via Patricia Trie
	// by having the leaf data being equal to keys.
	key := outputID.Bytes()
	return key, key
}
