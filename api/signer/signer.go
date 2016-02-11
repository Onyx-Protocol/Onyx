package signer

import (
	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

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

// SignLocalBlock signs b using the private key in s.
// It does not validate b.
//
// Note: not yet implemented.
func (s *Signer) SignLocalBlock(ctx context.Context, b *bc.Block) error {
	// TODO(kr): sign the block
	// 1. trust that the block is valid:
	//    - local generator will only generate valid blocks
	//    - it's a waste of resources to validate
	// 2. compute and return signature.
	panic("unimplemented")
}

// SignRemoteBlock validates b against the current blockchain
// and, if valid, signs b using the private key in s.
//
// SignRemoteBlock will never sign more than one block
// at any height. It ensures this invariant by storing
// the necessary state in its FC object.
//
// Note: not yet implemented.
func (s *Signer) SignRemoteBlock(ctx context.Context, b *bc.Block) error {
	// TODO(kr): sign the block
	// 1. validate block (except for sigscript eval)
	// 2. ensure we haven't signed any other block
	//    at this height (and never will)
	// 3. compute and return signature.
	panic("unimplemented")
}
