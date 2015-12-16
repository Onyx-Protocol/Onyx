package txdb

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/log"
	"chain/net/trace/span"
)

func topSort(ctx context.Context, txs []*bc.Tx) []*bc.Tx {
	if isTopSorted(ctx, txs) {
		return txs
	}

	log.Messagef(ctx, "set of %d txs not in topo order; sorting", len(txs))
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	if len(txs) == 1 {
		return txs
	}

	nodes := make(map[bc.Hash]*bc.Tx)
	for _, tx := range txs {
		nodes[tx.Hash] = tx
	}

	incomingEdges := make(map[bc.Hash]int)
	children := make(map[bc.Hash][]bc.Hash)
	for node, tx := range nodes {
		for _, in := range tx.Inputs {
			if prev := in.Previous.Hash; nodes[prev] != nil {
				if children[prev] == nil {
					children[prev] = make([]bc.Hash, 0, 1)
				}
				children[prev] = append(children[prev], node)
				incomingEdges[node]++
			}
		}
	}

	var s []bc.Hash
	for node := range nodes {
		if incomingEdges[node] == 0 {
			s = append(s, node)
		}
	}

	// https://en.wikipedia.org/wiki/Topological_sorting#Algorithms
	var l []*bc.Tx
	for len(s) > 0 {
		n := s[0]
		s = s[1:]
		l = append(l, nodes[n])

		for _, m := range children[n] {
			incomingEdges[m]--
			if incomingEdges[m] == 0 {
				delete(incomingEdges, m)
				s = append(s, m)
			}
		}
	}

	if len(incomingEdges) > 0 { // should be impossible
		panic("cyclical tx ordering")
	}

	return l
}

func isTopSorted(ctx context.Context, txs []*bc.Tx) bool {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	exists := make(map[bc.Hash]bool)
	seen := make(map[bc.Hash]bool)
	for _, tx := range txs {
		exists[tx.Hash] = true
	}
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if exists[in.Previous.Hash] && !seen[in.Previous.Hash] {
				return false
			}
		}
		seen[tx.Hash] = true
	}
	return true
}
