// Package txdb provides storage for Chain Protocol blockchain
// data structures.
package txdb

import (
	"context"
	"strconv"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

func ListenBlocks(ctx context.Context, dbURL string) (<-chan uint64, error) {
	listener, err := pg.NewListener(ctx, dbURL, "newblock")
	if err != nil {
		return nil, err
	}

	c := make(chan uint64)
	go func() {
		defer func() {
			listener.Close()
			close(c)
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case n := <-listener.Notify:
				height, err := strconv.ParseUint(n.Extra, 10, 64)
				if err != nil {
					log.Error(ctx, errors.Wrap(err, "parsing db notification payload"))
					return
				}
				c <- height
			}
		}
	}()

	return c, nil
}

// GetRawBlock queries the database for the block at the provided height.
// The block is returned as raw bytes.
func (s *Store) GetRawBlock(ctx context.Context, height uint64) ([]byte, error) {
	const q = `SELECT data FROM blocks WHERE height = $1`
	var block []byte
	err := s.db.QueryRow(ctx, q, height).Scan(&block)
	return block, errors.Wrap(err, "querying blocks from the db")
}
