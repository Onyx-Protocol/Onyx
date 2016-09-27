package txdb

import (
	"context"

	"github.com/golang/protobuf/proto"

	"chain/core/txdb/internal/storage"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/patricia"
	"chain/protocol/state"
)

func storeStateSnapshot(ctx context.Context, db pg.DB, snapshot *state.Snapshot, blockHeight uint64) error {
	var storedSnapshot storage.Snapshot
	err := patricia.Walk(snapshot.Tree, func(n *patricia.Node) error {
		hash := n.Hash()
		storedSnapshot.Nodes = append(storedSnapshot.Nodes, &storage.Snapshot_StateTreeNode{
			Key:  n.Key(),
			Leaf: n.IsLeaf(),
			Hash: hash[:],
		})
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "walking patricia tree")
	}

	storedSnapshot.Issuances = make([]*storage.Snapshot_Issuance, 0, len(snapshot.Issuances))
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
		var h bc.Hash
		copy(h[:], node.Hash)
		nodes = append(nodes, patricia.NewNode(node.Key, h, node.Leaf))
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
