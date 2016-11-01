package pin

import (
	"context"
	"sync"

	"chain/database/pg"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
)

type Store struct {
	DB pg.DB

	mu   sync.Mutex
	pins map[string]*Pin
}

func (ct *Store) Pin(name string) *Pin {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if ct.pins == nil {
		ct.pins = make(map[string]*Pin)
	}
	pin, ok := ct.pins[name]
	if !ok {
		pin = newPin(ct.DB, name, 0)
		ct.pins[name] = pin
	}
	return pin
}

func (ct *Store) ProcessBlocks(ctx context.Context, c *protocol.Chain, pinName string, cb func(context.Context, *bc.Block) error) {
	pin := ct.Pin(pinName)
	for {
		height := pin.Height()
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
			err = pin.RaiseTo(ctx, block.Height)
			if err != nil {
				log.Error(ctx, err)
			}
		case <-ctx.Done(): // leader deposed
			log.Error(ctx, ctx.Err())
			return
		}
	}
}

func (ct *Store) LoadAll(ctx context.Context) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.pins = make(map[string]*Pin)
	const q = `SELECT name, height FROM block_processors;`
	err := pg.ForQueryRows(ctx, ct.DB, q, func(name string, height uint64) {
		ct.pins[name] = newPin(ct.DB, name, height)
	})
	return err
}

type Pin struct {
	mu     sync.Mutex
	cond   sync.Cond
	height uint64

	db   pg.DB
	name string
}

func newPin(db pg.DB, name string, height uint64) *Pin {
	c := &Pin{db: db, name: name, height: height}
	c.cond.L = &c.mu
	return c
}

func (c *Pin) Height() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.height
}

func (c *Pin) RaiseTo(ctx context.Context, height uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	const q = `
		INSERT INTO block_processors (name, height) VALUES($1, $2)
		ON CONFLICT(name) DO UPDATE SET height=EXCLUDED.height
		WHERE block_processors.height<EXCLUDED.height
	`
	_, err := c.db.Exec(ctx, q, c.name, height)
	if err != nil {
		return err
	}
	if height > c.height {
		c.height = height
		c.cond.Broadcast()
	}
	return nil
}

func (c *Pin) WaitForHeight(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		for c.height < height {
			c.cond.Wait()
		}
		ch <- struct{}{}
	}()
	return ch
}
