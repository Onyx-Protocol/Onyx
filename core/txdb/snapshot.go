package txdb

import (
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"

	"chain/core/txdb/internal/storage"
	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
)

func storeStateSnapshot(ctx context.Context, db pg.DB, snapshot *state.Snapshot, blockHeight uint64) error {
	var storedSnapshot storage.Snapshot
	err := patricia.Walk(snapshot.Tree, func(n *patricia.Node) error {
		hash := n.Hash()
		var value []byte
		if v := n.Value(); !v.IsHash {
			value = n.Value().Bytes
		}

		storedSnapshot.Nodes = append(storedSnapshot.Nodes, &storage.Snapshot_StateTreeNode{
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

	for k, v := range snapshot.Issuances {
		hash := k
		storedSnapshot.Issuances = append(storedSnapshot.Issuances, &storage.Snapshot_Issuance{
			Hash:     hash[:],
			ExpiryMs: v,
		})
	}

	b, err := proto.Marshal(&storedSnapshot)
	if err != nil {
		return errors.Wrap(err, "marshaling state snapshot")
	}

	const insertQ = `
		INSERT INTO snapshots (height, data) VALUES($1, $2)
		ON CONFLICT (height) DO UPDATE SET data = $2
	`

	_, err = db.Exec(ctx, insertQ, blockHeight, b)
	return errors.Wrap(err, "writing state snapshot to database")
}

func getStateSnapshot(ctx context.Context, db pg.DB) (*state.Snapshot, uint64, error) {
	const q = `
		SELECT data, height FROM snapshots ORDER BY height DESC LIMIT 1
	`
	var (
		data           []byte
		height         uint64
		storedSnapshot storage.Snapshot
	)

	err := db.QueryRow(ctx, q).Scan(&data, &height)
	if err == sql.ErrNoRows {
		return state.Empty(), 0, nil
	} else if err != nil {
		return nil, height, errors.Wrap(err, "retrieving state snapshot blob")
	}

	err = proto.Unmarshal(data, &storedSnapshot)
	if err != nil {
		return nil, height, errors.Wrap(err, "unmarshaling state snapshot proto")
	}

	nodes := make([]*patricia.Node, 0, len(storedSnapshot.Nodes))
	for _, node := range storedSnapshot.Nodes {
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

	issuances := make(state.PriorIssuances, len(storedSnapshot.Issuances))
	for _, issuance := range storedSnapshot.Issuances {
		var hash bc.Hash
		copy(hash[:], issuance.Hash)
		issuances[hash] = issuance.ExpiryMs
	}

	snapshot := &state.Snapshot{
		Tree:      patricia.NewTree(nodes),
		Issuances: issuances,
	}
	return snapshot, height, nil
}
