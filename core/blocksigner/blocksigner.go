package blocksigner

import (
	"context"

	"chain/core/mockhsm"
	"chain/cos"
	"chain/cos/bc"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
)

// Signer validates and signs blocks.
type Signer struct {
	XPub *hd25519.XPub
	hsm  *mockhsm.HSM
	db   pg.DB
	fc   *cos.FC
}

// New returns a new Signer that validates blocks with fc and signs
// them with k.
//
// TODO(bobg): Create an HSM abstraction that allows HSM's other than
// the mockhsm to be used here.
func New(xpub *hd25519.XPub, hsm *mockhsm.HSM, db pg.DB, fc *cos.FC) *Signer {
	return &Signer{
		XPub: xpub,
		hsm:  hsm,
		db:   db,
		fc:   fc,
	}
}

// ComputeBlockSignature computes the signature for the block using
// the private key in s.  It does not validate the block.
func (s *Signer) ComputeBlockSignature(ctx context.Context, b *bc.Block) ([]byte, error) {
	hash := b.HashForSig()
	return s.hsm.Sign(ctx, s.XPub, nil, hash[:])
}

// SignBlock validates the given block against the current blockchain
// and, if valid, computes and returns a signature for the block.  It
// is used as the httpjson handler for /rpc/signer/sign-block.
//
// This function fails if this node has ever signed a block at the
// same height as b.
func (s *Signer) SignBlock(ctx context.Context, b *bc.Block) ([]byte, error) {
	fc := s.fc
	err := fc.WaitForBlock(ctx, b.Height-1)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for block at height %d", b.Height-1)
	}
	err = fc.ValidateBlockForSig(ctx, b)
	if err != nil {
		return nil, errors.Wrap(err, "validating block for signature")
	}
	err = lockBlockHeight(ctx, s.db, b)
	if err != nil {
		return nil, errors.Wrap(err, "lock block height")
	}
	return s.ComputeBlockSignature(ctx, b)
}

// lockBlockHeight records a signer's intention to sign a given block
// at a given height.  It's an error if a different block at the same
// height has previously been signed.
func lockBlockHeight(ctx context.Context, db pg.DB, b *bc.Block) error {
	const q = `
		INSERT INTO signed_blocks (block_height, block_hash)
		SELECT $1, $2
		    WHERE NOT EXISTS (SELECT 1 FROM signed_blocks
		                      WHERE block_height = $1 AND block_hash = $2)
	`
	_, err := db.Exec(ctx, q, b.Height, b.HashForSig())
	return err
}
