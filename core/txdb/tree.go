package txdb

import (
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"

	"chain/core/txdb/internal/storage"
	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
)

func storeStateTreeSnapshot(ctx context.Context, dbtx *sql.Tx, pt *patricia.Tree, blockHeight uint64) error {
	var snapshot storage.StateTree
	err := patricia.Walk(pt, func(n *patricia.Node) error {
		hash := n.Hash()
		var value []byte
		if v := n.Value(); !v.IsHash {
			value = n.Value().Bytes
		}

		snapshot.Nodes = append(snapshot.Nodes, &storage.StateTree_Node{
			Key:   n.Key(),
			Leaf:  n.IsLeaf(),
			Hash:  hash[:],
			Value: value,
		})
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "walking patricia tree")
	}

	b, err := proto.Marshal(&snapshot)
	if err != nil {
		return errors.Wrap(err, "marshaling state tree snapshot")
	}

	const insertQ = `
		INSERT INTO state_trees (height, data) VALUES($1, $2)
		ON CONFLICT (height) DO UPDATE SET data = $2
	`

	_, err = dbtx.Exec(ctx, insertQ, blockHeight, b)
	return errors.Wrap(err, "writing state tree to database")
}

func getStateTreeSnapshot(ctx context.Context, db pg.DB, blockHeight uint64) (*patricia.Tree, error) {
	const q = `
		SELECT data FROM state_trees WHERE height = $1
	`
	var data []byte
	var snapshot storage.StateTree

	err := db.QueryRow(ctx, q, blockHeight).Scan(&data)
	if err == sql.ErrNoRows {
		return patricia.NewTree(nil), nil
	} else if err != nil {
		return nil, errors.Wrap(err, "retrieving state tree blob")
	}

	err = proto.Unmarshal(data, &snapshot)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling state tree proto")
	}

	nodes := make([]*patricia.Node, 0, len(snapshot.Nodes))
	for _, node := range snapshot.Nodes {
		var v patricia.Valuer
		if len(node.Value) == 0 {
			var h bc.Hash
			copy(h[:], node.Hash)
			v = patricia.HashValuer(h)
		} else {
			v = patricia.BytesValuer(node.Value)
		}
		nodes = append(nodes, patricia.NewNode(node.Key, v, node.Leaf))
	}
	return patricia.NewTree(nodes), nil
}
