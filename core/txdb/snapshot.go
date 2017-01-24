package txdb

import (
	"context"

	"github.com/golang/protobuf/proto"

	"chain-stealth/core/txdb/internal/storage"
	"chain-stealth/database/pg"
	"chain-stealth/database/sql"
	"chain-stealth/errors"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/patricia"
	"chain-stealth/protocol/state"
)

// DecodeSnapshot decodes a snapshot from the Chain Core's binary,
// protobuf representation of the snapshot.
func DecodeSnapshot(data []byte) (*state.Snapshot, error) {
	var storedSnapshot storage.Snapshot
	err := proto.Unmarshal(data, &storedSnapshot)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling state snapshot proto")
	}

	reconstructTree := func(nodes []*storage.Snapshot_StateTreeNode) (*patricia.Tree, error) {
		leaves := make([]patricia.Leaf, len(nodes))
		for i, node := range nodes {
			leaves[i].Key = node.Key
			copy(leaves[i].Hash[:], node.Hash)
		}
		return patricia.Reconstruct(leaves)
	}

	tree1, err := reconstructTree(storedSnapshot.Nodes)
	if err != nil {
		return nil, errors.Wrap(err, "reconstructing state (v1) tree")
	}

	tree2, err := reconstructTree(storedSnapshot.Nodes2)
	if err != nil {
		return nil, errors.Wrap(err, "reconstructing state (v2) tree")
	}

	issuances := make(state.PriorIssuances, len(storedSnapshot.Issuances))
	for _, issuance := range storedSnapshot.Issuances {
		var hash bc.Hash
		copy(hash[:], issuance.Hash)
		issuances[hash] = issuance.ExpiryMs
	}

	return &state.Snapshot{
		Tree1:     tree1,
		Tree2:     tree2,
		Issuances: issuances,
	}, nil
}

func storeStateSnapshot(ctx context.Context, db pg.DB, snapshot *state.Snapshot, blockHeight uint64) error {
	var storedSnapshot storage.Snapshot
	err := patricia.Walk(snapshot.Tree1, func(l patricia.Leaf) error {
		storedSnapshot.Nodes = append(storedSnapshot.Nodes, &storage.Snapshot_StateTreeNode{
			Key:  l.Key,
			Hash: l.Hash[:],
		})
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "walking patricia tree")
	}

	err = patricia.Walk(snapshot.Tree2, func(l patricia.Leaf) error {
		storedSnapshot.Nodes2 = append(storedSnapshot.Nodes2, &storage.Snapshot_StateTreeNode{
			Key:  l.Key,
			Hash: l.Hash[:],
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
		data   []byte
		height uint64
	)

	err := db.QueryRow(ctx, q).Scan(&data, &height)
	if err == sql.ErrNoRows {
		return state.Empty(), 0, nil
	} else if err != nil {
		return nil, height, errors.Wrap(err, "retrieving state snapshot blob")
	}

	snapshot, err := DecodeSnapshot(data)
	if err != nil {
		return nil, height, errors.Wrap(err, "decoding snapshot")
	}
	return snapshot, height, nil
}

// getRawSnapshot returns the raw, protobuf-encoded snapshot data at the
// provided height.
func getRawSnapshot(ctx context.Context, db pg.DB, height uint64) (data []byte, err error) {
	const q = `SELECT data FROM snapshots WHERE height = $1`
	err = db.QueryRow(ctx, q, height).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	}
	return data, err
}
