package signer

import (
	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/crypto"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
)

// Signer validates and signs blocks.
type Signer struct {
	key *btcec.PrivateKey
	fc  *fedchain.FC
}

// New returns a new Signer
// that validates blocks with fc
// and signs them with k.
func New(k *btcec.PrivateKey, fc *fedchain.FC) *Signer {
	if k == nil {
		panic("signer key is unset")
	}
	return &Signer{
		key: k,
		fc:  fc,
	}
}

// ComputeBlockSignature computes the signature for the block using
// the private key in s.  It does not validate the block.
func (s *Signer) ComputeBlockSignature(b *bc.Block) (*btcec.Signature, error) {
	return fedchain.ComputeBlockSignature(b, s.key)
}

// SignBlock validates the given block against the current blockchain
// and, if valid, computes and returns a signature for the block.  It
// is used as the httpjson handler for /rpc/signer/sign-block.
//
// This function fails if this node has ever signed a block at the
// same height as that of the given block.  The heights of blocks it
// has signed are stored in the FC object.
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
