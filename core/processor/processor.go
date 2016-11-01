package processor

import (
	"context"
	"sync"

	"chain/database/pg"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
)

type CursorStore struct {
	DB pg.DB

	mu      sync.Mutex
	cursors map[string]*Cursor
}

func (ct *CursorStore) Cursor(name string) *Cursor {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if ct.cursors == nil {
		ct.cursors = make(map[string]*Cursor)
	}
	cursor, ok := ct.cursors[name]
	if !ok {
		cursor = newCursor(ct.DB, name, 0)
		ct.cursors[name] = cursor
	}
	return cursor
}

func (ct *CursorStore) ProcessBlocks(ctx context.Context, c *protocol.Chain, cursorName string, cb func(context.Context, *bc.Block) error) {
	cursor := ct.Cursor(cursorName)
	for {
		height := cursor.Height()
		select {
		case <-c.WaitForBlock(height + 1):
			block, err := c.GetBlock(ctx, height+1)
			if err != nil {
				log.Error(ctx, err)
				continue
			}
			err = cb(ctx, block)
			if err != nil {
				log.Error(ctx, err)
				continue
			}
			// This could cause issues, since it is not inside of a
			// database transaction.
			err = cursor.Increment(ctx)
			if err != nil {
				log.Error(ctx, err)
			}
		case <-ctx.Done(): // leader deposed
			log.Error(ctx, ctx.Err())
			break
		}
	}
}

func (ct *CursorStore) LoadAll(ctx context.Context) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.cursors = make(map[string]*Cursor)
	const q = `SELECT name, height FROM block_processors;`
	err := pg.ForQueryRows(ctx, ct.DB, q, func(name string, height uint64) {
		ct.cursors[name] = newCursor(ct.DB, name, height)
	})
	return err
}

type Cursor struct {
	mu     sync.Mutex
	cond   sync.Cond
	height uint64

	db   pg.DB
	name string
}

func newCursor(db pg.DB, name string, height uint64) *Cursor {
	c := &Cursor{db: db, name: name, height: height}
	c.cond.L = &c.mu
	return c
}

func (c *Cursor) Height() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.height
}

func (c *Cursor) Increment(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	const q = `
		INSERT INTO block_processors (name, height) VALUES($1, $2)
		ON CONFLICT(name) DO UPDATE SET height=EXCLUDED.height
	`
	_, err := c.db.Exec(ctx, q, c.name, c.height+1)
	if err != nil {
		return err
	}
	c.height++
	c.cond.Broadcast()
	return nil
}

func (c *Cursor) WaitForHeight(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		c.mu.Lock()
		for c.height < height {
			c.cond.Wait()
		}
		c.mu.Unlock()
		ch <- struct{}{}
	}()
	return ch
}
