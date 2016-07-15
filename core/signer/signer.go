package signer

import (
	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/crypto"
	"chain/database/pg"
	"chain/errors"
)

// Signer validates and signs blocks.
type Signer struct {
	key *btcec.PrivateKey
	db  pg.DB
	fc  *cos.FC
}

// New returns a new Signer
// that validates blocks with fc
// and signs them with k.
func New(k *btcec.PrivateKey, db pg.DB, fc *cos.FC) *Signer {
	if k == nil {
		panic("signer key is unset")
	}
	return &Signer{
		key: k,
		db:  db,
		fc:  fc,
	}
}

// ComputeBlockSignature computes the signature for the block using
// the private key in s.  It does not validate the block.
func (s *Signer) ComputeBlockSignature(b *bc.Block) (*btcec.Signature, error) {
	return cos.ComputeBlockSignature(b, s.key)
}

// SignBlock validates the given block against the current blockchain
// and, if valid, computes and returns a signature for the block.  It
// is used as the httpjson handler for /rpc/signer/sign-block.
//
// This function fails if this node has ever signed a block at the
// same height as b.
func (s *Signer) SignBlock(ctx context.Context, b *bc.Block) (*crypto.Signature, error) {
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
	signature, err := s.ComputeBlockSignature(b)
	if err != nil {
		return nil, err
	}
	return (*crypto.Signature)(signature), nil
}

// PublicKey gets the public key for the signer's private key.
func (s *Signer) PublicKey() *btcec.PublicKey {
	return s.key.PubKey()
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
