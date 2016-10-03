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
