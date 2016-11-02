package pin

import (
	"context"
	"strconv"
	"sync"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
)

type Store struct {
	DB pg.DB

	mu   sync.Mutex
	pins map[string]*Pin
}

func (s *Store) Pin(name string) *Pin {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pins == nil {
		s.pins = make(map[string]*Pin)
	}
	pin, ok := s.pins[name]
	if !ok {
		pin = newPin(s.DB, name, 0)
		s.pins[name] = pin
	}
	return pin
}

func (s *Store) ProcessBlocks(ctx context.Context, c *protocol.Chain, pinName string, cb func(context.Context, *bc.Block) error) {
	pin := s.Pin(pinName)
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

func (s *Store) LoadAll(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pins = make(map[string]*Pin)
	const q = `SELECT name, height FROM block_processors;`
	err := pg.ForQueryRows(ctx, s.DB, q, func(name string, height uint64) {
		s.pins[name] = newPin(s.DB, name, height)
	})
	return err
}

func (s *Store) WaitForAll(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		var pins []string
		s.mu.Lock()
		for name := range s.pins {
			pins = append(pins, name)
		}
		s.mu.Unlock()
		for _, name := range pins {
			<-s.Pin(name).WaitForHeight(height)
		}
		ch <- struct{}{}
	}()
	return ch
}

type Pin struct {
	mu     sync.Mutex
	cond   sync.Cond
	height uint64

	db   pg.DB
	name string
}

func newPin(db pg.DB, name string, height uint64) *Pin {
	p := &Pin{db: db, name: name, height: height}
	p.cond.L = &p.mu
	return p
}

func (p *Pin) Height() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.height
}

func (p *Pin) RaiseTo(ctx context.Context, height uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	const q = `
		INSERT INTO block_processors (name, height) VALUES($1, $2)
		ON CONFLICT(name) DO UPDATE SET height=EXCLUDED.height
		WHERE block_processors.height<EXCLUDED.height
	`
	const notifyQ = `SELECT pg_notify($1, $2)`
	_, err := p.db.Exec(ctx, q, p.name, height)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(ctx, notifyQ, "pin-"+p.name, height)
	if err != nil {
		return err
	}
	if height > p.height {
		p.height = height
		p.cond.Broadcast()
	}
	return nil
}

func (p *Pin) WaitForHeight(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		for p.height < height {
			p.cond.Wait()
		}
		ch <- struct{}{}
	}()
	return ch
}

func (p *Pin) Listen(ctx context.Context, dbURL string) {
	listener, err := pg.NewListener(ctx, dbURL, "pin-"+p.name)
	if err != nil {
		log.Error(ctx, err)
		return
	}

	go func() {
		defer func() {
			listener.Close()
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

				p.mu.Lock()
				if p.height < height {
					p.height = height
					p.cond.Broadcast()
				}
				p.mu.Unlock()
			}
		}
	}()

	return
}
