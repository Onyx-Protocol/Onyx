package txdb

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"

	"chain/core/txdb/internal/storage"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/patricia"
	"chain/protocol/state"
)

const maxSnapshotAge = 24 * time.Hour

// DecodeSnapshot decodes a snapshot from the Chain Core's binary,
// protobuf representation of the snapshot.
func DecodeSnapshot(data []byte) (*state.Snapshot, error) {
	var storedSnapshot storage.Snapshot
	err := proto.Unmarshal(data, &storedSnapshot)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling state snapshot proto")
	}

	leaves := make([]patricia.Leaf, len(storedSnapshot.Nodes))
	for i, node := range storedSnapshot.Nodes {
		leaves[i].Key = node.Key
		copy(leaves[i].Hash[:], node.Hash)
	}
	tree, err := patricia.Reconstruct(leaves)
	if err != nil {
		return nil, errors.Wrap(err, "reconstructing state tree")
	}

	issuances := make(state.PriorIssuances, len(storedSnapshot.Issuances))
	for _, issuance := range storedSnapshot.Issuances {
		var hash bc.Hash
		copy(hash[:], issuance.Hash)
		issuances[hash] = issuance.ExpiryMs
	}

	return &state.Snapshot{
		Tree:      tree,
		Issuances: issuances,
	}, nil
}

func storeStateSnapshot(ctx context.Context, db pg.DB, snapshot *state.Snapshot, blockHeight uint64) error {
	var storedSnapshot storage.Snapshot
	err := patricia.Walk(snapshot.Tree, func(l patricia.Leaf) error {
		storedSnapshot.Nodes = append(storedSnapshot.Nodes, &storage.Snapshot_StateTreeNode{
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
	if err != nil {
		return errors.Wrap(err, "writing state snapshot to database")
	}

	const deleteQ = `DELETE FROM snapshots WHERE timestamp < $1`
	_, err = db.Exec(ctx, deleteQ, time.Now().Add(-maxSnapshotAge))
	return errors.Wrap(err, "deleting old snapshots")
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
