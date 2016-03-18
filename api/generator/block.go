package generator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec"

	"golang.org/x/net/context"

	"chain/api/rpcclient"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
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

// MakeBlock creates a new bc.Block and updates the txpool/utxo state.
func MakeBlock(ctx context.Context) (*bc.Block, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	b, prevBlock, err := fc.GenerateBlock(ctx, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "generate")
	}
	if len(b.Transactions) == 0 {
		return nil, nil // don't bother making an empty block
	}

	err = GetAndAddBlockSignatures(ctx, b, prevBlock)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}

	err = fc.AddBlock(ctx, b)
	if err != nil {
		return nil, errors.Wrap(err, "apply")
	}
	return b, nil
}

func GetAndAddBlockSignatures(ctx context.Context, b, prevBlock *bc.Block) error {
	pubkeys, nrequired, err := txscript.ParseMultiSigScript(prevBlock.OutputScript)
	if err != nil {
		return errors.Wrap(err, "parsing prevblock output script")
	}

	signersConfigured := len(remoteSigners)
	if localSigner != nil {
		signersConfigured++
	}
	if signersConfigured < nrequired {
		return ErrTooFewSigners
	}

	signersByPubkey := make(map[string]*RemoteSigner, signersConfigured)
	for _, remoteSigner := range remoteSigners {
		signersByPubkey[pubkeyString(remoteSigner.Key)] = remoteSigner
	}
	if localSigner != nil {
		signersByPubkey[pubkeyString(localSigner.PublicKey())] = nil
	}

	type response struct {
		signature *btcec.Signature
		signer    *RemoteSigner
		err       error
		pos       int
	}

	var serializedBlock []byte

	responses := make(chan *response, len(pubkeys))

	var nrequests int

	for i, pubkey := range pubkeys {
		signer, ok := signersByPubkey[pubkeyString(pubkey)]
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
				r.signature, r.err = localSigner.ComputeBlockSignature(b)
			} else {
				r.signature, r.err = rpcclient.GetSignatureForSerializedBlock(ctx, signer.URL.String(), serializedBlock)
			}
			responses <- r
		}(i)
		nrequests++
	}

	ready := make([]*btcec.Signature, nrequests)
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
			signatures := make([]*btcec.Signature, 0, nready)
			for _, r := range ready {
				if r != nil {
					signatures = append(signatures, r)
				}
			}
			return fedchain.AddSignaturesToBlock(b, signatures)
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

func pubkeyString(pubkey *btcec.PublicKey) string {
	return string(pubkey.SerializeCompressed())
}
