package generator

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/net/trace/span"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vmutil"
)

// errTooFewSigners is returned when a block-signing attempt finds
// that not enough signers are configured for the number of
// signatures required.
var errTooFewSigners = errors.New("too few signers")

// makeBlock generates a new bc.Block, collects the required signatures
// and commits the block to the blockchain.
func (g *generator) makeBlock(ctx context.Context) (*bc.Block, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	b, s, err := g.chain.GenerateBlock(ctx, g.latestBlock, g.latestSnapshot, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "generate")
	}
	if len(b.Transactions) == 0 {
		return nil, nil // don't bother making an empty block
	}
	err = g.savePendingBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	return g.commitBlock(ctx, b, s)
}

func (g *generator) commitBlock(ctx context.Context, b *bc.Block, s *state.Snapshot) (*bc.Block, error) {
	err := g.getAndAddBlockSignatures(ctx, b, g.latestBlock)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}

	err = g.chain.CommitBlock(ctx, b, s)
	if err != nil {
		return nil, errors.Wrap(err, "commit")
	}

	g.latestBlock = b
	g.latestSnapshot = s
	return b, nil
}

func (g *generator) getAndAddBlockSignatures(ctx context.Context, b, prevBlock *bc.Block) error {
	if prevBlock == nil && b.Height == 1 {
		return nil // no signatures needed for initial block
	}

	pubkeys, quorum, err := vmutil.ParseBlockMultiSigProgram(prevBlock.ConsensusProgram)
	if err != nil {
		return errors.Wrap(err, "parsing prevblock output script")
	}
	if len(g.signers) < quorum {
		return errTooFewSigners
	}

	hashForSig := b.HashForSig()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	goodSigs := make([][]byte, len(pubkeys))
	replies := make([][]byte, len(g.signers))
	done := make(chan int, len(g.signers))
	for i, signer := range g.signers {
		go getSig(ctx, signer, b, &replies[i], i, done)
	}

	nready := 0
	for i := 0; i < len(g.signers) && nready < quorum; i++ {
		sig := replies[<-done]
		if sig == nil {
			continue
		}
		k := indexKey(pubkeys, hashForSig[:], sig)
		if k >= 0 && goodSigs[k] == nil {
			goodSigs[k] = sig
			nready++
		} else if k < 0 {
			log.Write(ctx, "error", "invalid signature", "block", b.Hash(), "signature", sig)
		}
	}

	if nready < quorum {
		return fmt.Errorf("got %d of %d needed signatures", nready, quorum)
	}
	b.Witness = nonNilSigs(goodSigs)
	return nil
}

func indexKey(keys []ed25519.PublicKey, msg, sig []byte) int {
	for i, key := range keys {
		if ed25519.Verify(key, msg, sig) {
			return i
		}
	}
	return -1
}

func getSig(ctx context.Context, signer BlockSigner, b *bc.Block, sig *[]byte, i int, done chan int) {
	var err error
	*sig, err = signer.SignBlock(ctx, b)
	if err != nil {
		log.Write(ctx, "error", err, "signer", signer)
	}
	done <- i
}

func nonNilSigs(a [][]byte) (b [][]byte) {
	for _, p := range a {
		if p != nil {
			b = append(a, p)
		}
	}
	return b
}

// getPendingBlock retrieves the generated, uncomitted block if it exists.
func (g *generator) getPendingBlock(ctx context.Context) (*bc.Block, error) {
	const q = `SELECT data FROM generator_pending_block`
	var block bc.Block
	err := pg.QueryRow(ctx, q).Scan(&block)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "retrieving generated pending block query")
	}
	return &block, nil
}

// savePendingBlock persists a pending, uncommitted block to the database.
// The generator should save a pending block *before* asking signers to
// sign the block.
func (g *generator) savePendingBlock(ctx context.Context, b *bc.Block) error {
	const q = `
		INSERT INTO generator_pending_block (data) VALUES($1)
		ON CONFLICT (singleton) DO UPDATE SET data = $1;
	`
	_, err := pg.Exec(ctx, q, b)
	return errors.Wrap(err, "generator_pending_block insert query")
}

// SaveInitialBlock saves b as the generator's pending block.
// Block b must have height 1.
// It is an error to save an initial block after other blocks
// have been generated.
func SaveInitialBlock(ctx context.Context, db pg.DB, b *bc.Block) error {
	if b.Height != 1 {
		return errors.Wrap(fmt.Errorf("generator: bad initial block height %d", b.Height))
	}
	// the insert is meant to fail if a block has ever been generated before
	const q = `INSERT INTO generator_pending_block (data) values ($1)`
	_, err := db.Exec(ctx, q, b)
	return errors.Wrap(err)
}
