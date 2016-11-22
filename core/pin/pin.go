package pin

import (
	"context"
	"sort"
	"strconv"
	"sync"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
)

const processorWorkers = 10

type Store struct {
	db pg.DB

	mu   sync.Mutex
	cond sync.Cond
	pins map[string]*pin
}

func NewStore(db pg.DB) *Store {
	s := &Store{
		db:   db,
		pins: make(map[string]*pin),
	}
	s.cond.L = &s.mu
	return s
}

func (s *Store) ProcessBlocks(ctx context.Context, c *protocol.Chain, pinName string, cb func(context.Context, *bc.Block) error) {
	p := <-s.pin(pinName)
	height := p.getHeight()
	for {
		select {
		case <-ctx.Done(): // leader deposed
			log.Error(ctx, ctx.Err())
			return
		case <-c.BlockWaiter(height + 1):
			select {
			case <-ctx.Done():
				log.Error(ctx, ctx.Err())
				return
			case p.sem <- true:
				go p.processBlock(ctx, c, height+1, cb)
				height++
			}
		}
	}
}

func (s *Store) CreatePin(ctx context.Context, name string, height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pins[name]; ok {
		return nil
	}
	const q = `
		INSERT INTO block_processors (name, height) VALUES($1, $2)
		ON CONFLICT(name) DO NOTHING;
	`
	_, err := s.db.Exec(ctx, q, name, height)
	if err != nil {
		return errors.Wrap(err)
	}
	s.pins[name] = newPin(s.db, name, height)
	s.cond.Broadcast()
	return nil
}

func (s *Store) Height(name string) uint64 {
	p := <-s.pin(name)
	return p.getHeight()
}

func (s *Store) LoadAll(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	const q = `SELECT name, height FROM block_processors;`
	err := pg.ForQueryRows(ctx, s.db, q, func(name string, height uint64) {
		s.pins[name] = newPin(s.db, name, height)
	})
	s.cond.Broadcast()
	return err
}

func (s *Store) pin(name string) <-chan *pin {
	ch := make(chan *pin, 1)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for s.pins[name] == nil {
			s.cond.Wait()
		}
		ch <- s.pins[name]
	}()
	return ch
}

func (s *Store) PinWaiter(pinName string, height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	p := <-s.pin(pinName)
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

func (s *Store) AllWaiter(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		var pins []string
		s.mu.Lock()
		for name := range s.pins {
			pins = append(pins, name)
		}
		s.mu.Unlock()
		for _, name := range pins {
			<-s.PinWaiter(name, height)
		}
		ch <- struct{}{}
	}()
	return ch
}

func (s *Store) Listen(ctx context.Context, pinName, dbURL string) {
	listener, err := pg.NewListener(ctx, dbURL, "pin-"+pinName)
	if err != nil {
		log.Error(ctx, err)
		return
	}
	go func() {
		defer listener.Close()

		var p *pin

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

				if p == nil {
					s.mu.Lock()
					var ok bool
					p, ok = s.pins[pinName]
					if !ok {
						p = newPin(s.db, pinName, height)
						s.pins[pinName] = p
						s.cond.Broadcast()
					}
					s.mu.Unlock()
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

type pin struct {
	mu        sync.Mutex
	cond      sync.Cond
	height    uint64
	completed []uint64

	db   pg.DB
	name string
	sem  chan bool
}

func newPin(db pg.DB, name string, height uint64) *pin {
	p := &pin{db: db, name: name, height: height, sem: make(chan bool, processorWorkers)}
	p.cond.L = &p.mu
	return p
}

func (p *pin) getHeight() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.height
}

func (p *pin) processBlock(ctx context.Context, c *protocol.Chain, height uint64, cb func(context.Context, *bc.Block) error) {
	defer func() { <-p.sem }()
	for {
		block, err := c.GetBlock(ctx, height)
		if err != nil {
			log.Error(ctx, err)
			continue
		}
		err = cb(ctx, block)
		if err != nil {
			log.Error(ctx, err)
			continue
		}
		err = p.complete(ctx, block.Height)
		if err != nil {
			log.Error(ctx, err)
		}
		break
	}
}

func (p *pin) complete(ctx context.Context, height uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.completed = append(p.completed, height)
	sort.Sort(uint64s(p.completed))

	var (
		max = p.height
		i   int
	)
	for i = 0; i < len(p.completed); i++ {
		if p.completed[i] <= max {
			continue
		} else if p.completed[i] > max+1 {
			break
		}
		max = p.completed[i]
	}

	if max == p.height {
		return nil
	}

	const q = `UPDATE block_processors SET height=$1 WHERE height<$1 AND name=$2`
	_, err := p.db.Exec(ctx, q, max, p.name)
	if err != nil {
		return err
	}

	const notifyQ = `SELECT pg_notify($1, $2)`
	_, err = p.db.Exec(ctx, notifyQ, "pin-"+p.name, max)
	if err != nil {
		return err
	}

	p.completed = p.completed[i:]
	p.height = max
	p.cond.Broadcast()

	return nil
}

type uint64s []uint64

func (a uint64s) Len() int           { return len(a) }
func (a uint64s) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a uint64s) Less(i, j int) bool { return a[i] < a[j] }
