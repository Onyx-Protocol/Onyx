package txdb

import (
	"context"
	"strconv"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

// New creates a Store and Pool backed by the txdb with the provided
// db handle.
func New(db pg.DB) (*Store, *Pool) {
	return NewStore(db), NewPool(db)
}

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

// GetRawBlocks queries the database for blocks after the provided height,
// returning up to count blocks. The blocks are returned as raw bytes.
func (s *Store) GetRawBlocks(ctx context.Context, afterHeight uint64, count int) ([][]byte, error) {
	const q = `SELECT data FROM blocks WHERE height > $1 ORDER BY height LIMIT `
	var blocks [][]byte
	err := pg.ForQueryRows(ctx, s.db, q+strconv.Itoa(count), afterHeight, func(rawBlock []byte) {
		blocks = append(blocks, rawBlock)
	})
	return blocks, errors.Wrap(err, "querying blocks from the db")
}
