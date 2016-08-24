package generator

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"chain/core/rpcclient"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/vmutil"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
	"chain/net/trace/span"
)

var (
	// ErrTooFewSigners is returned when a block-signing attempt finds
	// that not enough signers are configured for the number of
	// signatures required.
	ErrTooFewSigners = errors.New("too few signers")

	// ErrUnknownPubkey is returned when a block-signing attempt finds
	// an unrecognized public key in the output script of the previous
	// block.
	ErrUnknownPubkey = errors.New("unknown block pubkey")
)

// MakeBlock generates a new bc.Block, collects the required signatures
// and commits the block to the blockchain.
func (g *Generator) MakeBlock(ctx context.Context) (*bc.Block, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	b, err := g.FC.GenerateBlock(ctx, g.latestBlock, g.latestSnapshot, time.Now())
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
	return g.commitBlock(ctx, b)
}

func (g *Generator) commitBlock(ctx context.Context, b *bc.Block) (*bc.Block, error) {
	err := g.GetAndAddBlockSignatures(ctx, b, g.latestBlock)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}

	// Apply the block to get the state snapshot and commit it.
	snapshot, err := g.FC.ValidateBlock(ctx, g.latestSnapshot, g.latestBlock, b)
	if err != nil {
		return nil, errors.Wrap(err, "apply")
	}
	err = g.FC.CommitBlock(ctx, b, snapshot)
	if err != nil {
		return nil, errors.Wrap(err, "commit")
	}

	g.latestBlock = b
	g.latestSnapshot = snapshot
	return b, nil
}

func (g *Generator) GetAndAddBlockSignatures(ctx context.Context, b, prevBlock *bc.Block) error {
	if prevBlock == nil && b.Height == 1 {
		return nil // no signatures needed for initial block
	}

	pubkeys, nrequired, err := vmutil.ParseBlockMultiSigScript(prevBlock.ConsensusProgram)
	if err != nil {
		return errors.Wrap(err, "parsing prevblock output script")
	}
	if nrequired == 0 {
		return nil // no signatures needed
	}

	signersConfigured := len(g.RemoteSigners)
	if g.LocalSigner != nil {
		signersConfigured++
	}
	if signersConfigured < nrequired {
		return ErrTooFewSigners
	}

	signersByPubkey := make(map[string]*RemoteSigner, signersConfigured)
	for _, remoteSigner := range g.RemoteSigners {
		signersByPubkey[keystr(remoteSigner.Key)] = remoteSigner
	}
	if g.LocalSigner != nil {
		signersByPubkey[keystr(g.LocalSigner.XPub.Key)] = nil
	}

	type response struct {
		signature []byte
		signer    *RemoteSigner
		err       error
		pos       int
	}

	var (
		nrequests       int
		serializedBlock []byte
		responses       = make(chan *response, len(pubkeys))
	)
	for i, pubkey := range pubkeys {
		signer, ok := signersByPubkey[keystr(pubkey)]
		if !ok {
			return ErrUnknownPubkey
		}

		if signer != nil && serializedBlock == nil {
			// Optimization: serialize the block just once instead of in N
			// goroutines (and not at all if only using a local signer).
			serializedBlock, err = json.Marshal(b)
			if err != nil {
				return errors.Wrap(err, "serializing block")
			}
		}

		go func(pos int) {
			r := &response{
				signer: signer,
				pos:    pos,
			}
			if signer == nil {
				r.signature, r.err = g.LocalSigner.ComputeBlockSignature(ctx, b)
			} else {
				r.signature, r.err = rpcclient.GetSignatureForSerializedBlock(ctx, signer.URL.String(), serializedBlock)
			}
			responses <- r
		}(i)
		nrequests++
	}

	ready := make([][]byte, nrequests)
	var nready int
	var errResponses []*response

	for i := 0; i < nrequests; i++ {
		response := <-responses
		if response.err != nil {
			errResponses = append(errResponses, response)
		}
		ready[response.pos] = response.signature
		nready++
		if nready >= nrequired {
			signatures := make([][]byte, 0, nready)
			for _, r := range ready {
				if r != nil {
					signatures = append(signatures, r)
				}
			}
			cos.AddSignaturesToBlock(b, signatures)
			return nil
		}
	}

	// Didn't get enough signatures
	errMsg := fmt.Sprintf("got %d of %d needed signature(s)", nready, nrequired)
	for _, errResponse := range errResponses {
		var addr string
		if errResponse.signer == nil {
			addr = "local"
		} else {
			addr = errResponse.signer.URL.String()
		}
		errMsg += fmt.Sprintf(" [%s: %s]", addr, errResponse.err)
	}
	return errors.New(errMsg)
}

func keystr(k ed25519.PublicKey) string {
	return hex.EncodeToString(hd25519.PubBytes(k))
}

// getPendingBlock retrieves the generated, uncomitted block if it exists.
func (g *Generator) getPendingBlock(ctx context.Context) (*bc.Block, error) {
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
func (g *Generator) savePendingBlock(ctx context.Context, b *bc.Block) error {
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
