package txbuilder

import (
	"context"
	"encoding/json"

	"chain/crypto/ed25519/chainkd"
	"chain/crypto/sha3pool"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/vm"
)

type (
	SignatureWitness struct {
		// Quorum is the number of signatures required.
		Quorum int `json:"quorum"`

		// Keys are the identities of the keys to sign with.
		Keys []keyID `json:"keys"`

		// Program is the predicate part of the signature program, whose hash is what gets
		// signed. If empty, it is computed during Sign from the outputs
		// and the current input of the transaction.
		Program chainjson.HexBytes `json:"program"`

		// Sigs are signatures of Program made from each of the Keys
		// during Sign.
		Sigs []chainjson.HexBytes `json:"signatures"`
	}

	keyID struct {
		XPub           chainkd.XPub         `json:"xpub"`
		DerivationPath []chainjson.HexBytes `json:"derivation_path"`
	}
)

var ErrEmptyProgram = errors.New("empty signature program")

// Sign populates sw.Sigs with as many signatures of the predicate in
// sw.Program as it can from the overlapping set of keys in sw.Keys
// and xpubs.
//
// If sw.Program is empty, it is populated with an _inferred_ predicate:
// a program committing to aspects of the current
// transaction. Specifically, the program commits to:
//  - the mintime and maxtime of the transaction (if non-zero)
//  - the outputID and (if non-empty) reference data of the current input
//  - the assetID, amount, control program, and (if non-empty) reference data of each output.
func (sw *SignatureWitness) sign(ctx context.Context, tpl *Template, index uint32, xpubs []chainkd.XPub, signFn SignFunc) error {
	// Compute the predicate to sign. This is either a
	// txsighash program if tpl.AllowAdditional is false (i.e., the tx is complete
	// and no further changes are allowed) or a program enforcing
	// constraints derived from the existing outputs and current input.
	if len(sw.Program) == 0 {
		sw.Program = buildSigProgram(tpl, tpl.SigningInstructions[index].Position)
		if len(sw.Program) == 0 {
			return ErrEmptyProgram
		}
	}
	if len(sw.Sigs) < len(sw.Keys) {
		// Each key in sw.Keys may produce a signature in sw.Sigs. Make
		// sure there are enough slots in sw.Sigs and that we preserve any
		// sigs already present.
		newSigs := make([]chainjson.HexBytes, len(sw.Keys))
		copy(newSigs, sw.Sigs)
		sw.Sigs = newSigs
	}
	var h [32]byte
	sha3pool.Sum256(h[:], sw.Program)
	for i, keyID := range sw.Keys {
		if len(sw.Sigs[i]) > 0 {
			// Already have a signature for this key
			continue
		}
		var found bool
		for _, xpub := range xpubs {
			if keyID.XPub == xpub {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		path := make([]([]byte), len(keyID.DerivationPath))
		for i, p := range keyID.DerivationPath {
			path[i] = p
		}
		sigBytes, err := signFn(ctx, keyID.XPub, path, h)
		if err != nil {
			return errors.WithDetailf(err, "computing signature %d", i)
		}
		sw.Sigs[i] = sigBytes
	}
	return nil
}

func (sw SignatureWitness) materialize(args *[][]byte) error {
	// This is the value of N for the CHECKPREDICATE call. The code
	// assumes that everything already in the arg list before this call
	// to Materialize is input to the signature program, so N is
	// len(*args).
	*args = append(*args, vm.Int64Bytes(int64(len(*args))))

	var nsigs int
	for i := 0; i < len(sw.Sigs) && nsigs < sw.Quorum; i++ {
		if len(sw.Sigs[i]) > 0 {
			*args = append(*args, sw.Sigs[i])
			nsigs++
		}
	}
	*args = append(*args, sw.Program)
	return nil
}

func (sw SignatureWitness) MarshalJSON() ([]byte, error) {
	obj := struct {
		Type   string               `json:"type"`
		Quorum int                  `json:"quorum"`
		Keys   []keyID              `json:"keys"`
		Sigs   []chainjson.HexBytes `json:"signatures"`
	}{
		Type:   "signature",
		Quorum: sw.Quorum,
		Keys:   sw.Keys,
		Sigs:   sw.Sigs,
	}
	return json.Marshal(obj)
}
