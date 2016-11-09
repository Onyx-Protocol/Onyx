package pin

import (
	"context"
	"database/sql"
	"strconv"
	"sync"
	"time"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
)

const queueTimeout = time.Second * 5

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

func (s *Store) ProcessBlock(ctx context.Context, c *protocol.Chain, processID, name string, f func(context.Context, *bc.Block) error) error {
	height, err := s.HeightToProcess(ctx, processID, name)
	if err != nil {
		return err
	}
	defer s.Release(ctx, processID, name, height)
	block, err := c.GetBlock(ctx, height)
	if err != nil {
		return err
	}
	err = f(ctx, block)
	if err != nil {
		return err
	}
	return s.Complete(ctx, processID, name, height)
}

func (s *Store) HeightToProcess(ctx context.Context, processID, name string) (uint64, error) {
	p := <-s.pin(name)
	<-p.waitForQueue()
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-p.queueListener:
		}
		const q = `
			WITH block AS (
				SELECT name, height FROM block_processor_queue
				WHERE name=$1 AND (held_by IS NULL OR held_at<$2)
				FOR UPDATE SKIP LOCKED
				LIMIT 1
			)
			UPDATE block_processor_queue bpq
			SET held_by=$3, held_at=$4
			FROM block
			WHERE (bpq.name, bpq.height) = (block.name, block.height)
			RETURNING bpq.height
		`
		var height uint64
		err := s.db.QueryRow(ctx, q, name, time.Now().Add(-queueTimeout), processID, time.Now()).Scan(&height)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return 0, errors.Wrap(err)
		}
		return height, nil
	}
}

func (s *Store) Complete(ctx context.Context, processID, name string, height uint64) error {
	p := <-s.pin(name)
	const deleteQ = `
		DELETE FROM block_processor_queue WHERE (name, height) = ($1, $2) AND held_by=$3
	`
	_, err := s.db.Exec(ctx, deleteQ, name, height, processID)
	if err != nil {
		return errors.Wrap(err)
	}

	const heightQ = `
		SELECT
			COALESCE((SELECT MIN(height) FROM block_processor_queue WHERE name=$1), 0) AS min_queued,
			queued_height FROM block_processors WHERE name=$1
	`
	var minQueued, maxQueued uint64
	err = s.db.QueryRow(ctx, heightQ, name).Scan(&minQueued, &maxQueued)
	if err != nil {
		return errors.Wrap(err)
	}
	var processedHeight uint64
	if minQueued == 0 { // no queued items
		processedHeight = maxQueued
	} else {
		processedHeight = minQueued - 1
	}

	return p.raiseTo(ctx, processedHeight)
}

// Release frees a held processor job in the queue.
// It can
func (s *Store) Release(ctx context.Context, processID, name string, height uint64) error {
	const q = `
		WITH updated AS (
			UPDATE block_processor_queue
			SET held_by=NULL, held_at=NULL
			WHERE (name, height)=($1, $2) AND held_by=$3
			RETURNING name, height
		)
		SELECT pg_notify(name, height) FROM updated;
	`
	_, err := s.db.Exec(ctx, q, name, height)
	return errors.Wrap(err)
}

func (s *Store) QueueBlocks(ctx context.Context, c *protocol.Chain, name string) {
	p := <-s.pin(name)
	for {
		height := p.getQueuedHeight()
		select {
		case <-ctx.Done(): // leader deposed
			log.Error(ctx, ctx.Err())
			return
		case <-c.WaitForBlock(height + 1):
			err := s.addToQueue(ctx, name, height+1)
			if err != nil {
				log.Error(ctx, err)
				continue
			}
		}
	}
}

func (s *Store) addToQueue(ctx context.Context, name string, height uint64) error {
	const insertQ = `
		INSERT INTO block_processor_queue (name, height) VALUES($1, $2)
		ON CONFLICT(name, height) DO NOTHING;
	`
	_, err := s.db.Exec(ctx, insertQ, name, height)
	if err != nil {
		return errors.Wrap(err)
	}
	p := <-s.pin(name)
	return p.raiseQueuedHeight(ctx, height)
}

func (s *Store) CreatePin(ctx context.Context, name string, height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pins[name]; ok {
		return nil
	}
	const q = `
		INSERT INTO block_processors (name, height, queued_height) VALUES($1, $2, $3)
		ON CONFLICT(name) DO NOTHING;
	`
	_, err := s.db.Exec(ctx, q, name, height, height)
	if err != nil {
		return errors.Wrap(err)
	}
	s.pins[name] = newPin(s.db, name, height)
	s.cond.Broadcast()
	return nil
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

func (s *Store) WaitForPin(name string, height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	p := <-s.pin(name)
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
			<-s.WaitForPin(name, height)
		}
		ch <- struct{}{}
	}()
	return ch
}

func (s *Store) Listen(ctx context.Context, name, dbURL string) {
	listener, err := pg.NewListener(ctx, dbURL, "pin-"+name)
	if err != nil {
		log.Error(ctx, err)
		return
	}
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
				p, ok = s.pins[name]
				if !ok {
					p = newPin(s.db, name, height)
					s.pins[name] = p
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
}

func (s *Store) ListenQueue(ctx context.Context, name, dbURL string) {
	p := <-s.pin(name)
	listener, err := pg.NewListener(ctx, dbURL, "pinqueue-"+p.name)
	if err != nil {
		log.Error(ctx, err)
		return
	}
	p.mu.Lock()
	p.queueListener = make(chan struct{}, 1)
	p.cond.Broadcast()
	p.mu.Unlock()
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-listener.Notify:
			p.queueListener <- struct{}{}
		}
	}
}

type pin struct {
	mu            sync.Mutex
	cond          sync.Cond
	height        uint64
	queuedHeight  uint64
	queueListener chan struct{}

	db   pg.DB
	name string
}

func newPin(db pg.DB, name string, height uint64) *pin {
	p := &pin{db: db, name: name, height: height, queuedHeight: height}
	p.cond.L = &p.mu
	return p
}

func (p *pin) waitForQueue() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		for p.queueListener == nil {
			p.cond.Wait()
		}
		ch <- struct{}{}
	}()
	return ch
}

func (p *pin) getHeight() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.height
}

func (p *pin) getQueuedHeight() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.queuedHeight
}

func (p *pin) raiseQueuedHeight(ctx context.Context, height uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	const q = `UPDATE block_processors SET queued_height=$1 WHERE queued_height<$1 AND name=$2`
	_, err := p.db.Exec(ctx, q, height, p.name)
	if err != nil {
		return err
	}

	const notifyQ = `SELECT pg_notify($1, $2)`
	_, err = p.db.Exec(ctx, notifyQ, "pinqueue-"+p.name, height)
	if err != nil {
		return err
	}

	if height > p.queuedHeight {
		p.queuedHeight = height
		go func() {
			<-p.waitForQueue()
			p.queueListener <- struct{}{}
		}()
	}
	return nil
}

func (p *pin) raiseTo(ctx context.Context, height uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	const q = `UPDATE block_processors SET height=$1 WHERE height<$1 AND name=$2`
	_, err := p.db.Exec(ctx, q, height, p.name)
	if err != nil {
		return err
	}

	const notifyQ = `SELECT pg_notify($1, $2)`
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
