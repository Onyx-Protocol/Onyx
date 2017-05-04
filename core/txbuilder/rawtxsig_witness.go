package txbuilder

import (
	"context"
	"encoding/json"

	"chain/crypto/ed25519/chainkd"
	chainjson "chain/encoding/json"
	"chain/errors"
)

// TODO(bobg): most of the code here is duplicated from
// signature_witness.go and needs refactoring.

// RawTxSigWitness is like SignatureWitness but doesn't involve
// signature programs.
type RawTxSigWitness struct {
	Quorum int                  `json:"quorum"`
	Keys   []keyID              `json:"keys"`
	Sigs   []chainjson.HexBytes `json:"signatures"`
}

func (sw *RawTxSigWitness) sign(ctx context.Context, tpl *Template, index uint32, xpubs []chainkd.XPub, signFn SignFunc) error {
	if len(sw.Sigs) < len(sw.Keys) {
		// Each key in sw.Keys may produce a signature in sw.Sigs. Make
		// sure there are enough slots in sw.Sigs and that we preserve any
		// sigs already present.
		newSigs := make([]chainjson.HexBytes, len(sw.Keys))
		copy(newSigs, sw.Sigs)
		sw.Sigs = newSigs
	}
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
		sigBytes, err := signFn(ctx, keyID.XPub, path, tpl.Hash(index).Byte32())
		if err != nil {
			return errors.WithDetailf(err, "computing signature %d", i)
		}
		sw.Sigs[i] = sigBytes
	}
	return nil
}

func (sw RawTxSigWitness) materialize(args *[][]byte) error {
	var nsigs int
	for i := 0; i < len(sw.Sigs) && nsigs < sw.Quorum; i++ {
		if len(sw.Sigs[i]) > 0 {
			*args = append(*args, sw.Sigs[i])
			nsigs++
		}
	}
	return nil
}

func (sw RawTxSigWitness) MarshalJSON() ([]byte, error) {
	obj := struct {
		Type   string               `json:"type"`
		Quorum int                  `json:"quorum"`
		Keys   []keyID              `json:"keys"`
		Sigs   []chainjson.HexBytes `json:"signatures"`
	}{
		Type:   "raw_tx_signature",
		Quorum: sw.Quorum,
		Keys:   sw.Keys,
		Sigs:   sw.Sigs,
	}
	return json.Marshal(obj)
}
