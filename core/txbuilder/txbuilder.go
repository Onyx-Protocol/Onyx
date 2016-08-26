package txbuilder

import (
	"bytes"
	"context"
	"fmt"

	"chain/crypto/ed25519/hd25519"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

var (
	ErrBadRefData       = errors.New("transaction reference data does not match previous template's reference data")
	ErrBadTxInputIdx    = errors.New("unsigned tx missing input")
	ErrBadSigScriptComp = errors.New("invalid signature script component")
	ErrMissingSig       = errors.New("missing signature in template")
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and destinations unsatisfied.
// Build partners then satisfy and consume inputs and destinations.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, tx *bc.TxData, actions []Action, ref json.Map) (*Template, error) {
	if tx == nil {
		tx = &bc.TxData{
			Version: bc.CurrentTransactionVersion,
		}
	}

	if len(ref) != 0 {
		if len(tx.ReferenceData) != 0 && !bytes.Equal(tx.ReferenceData, ref) {
			return nil, errors.Wrap(ErrBadRefData)
		}

		tx.ReferenceData = ref
	}

	var tplInputs []*Input
	for i, action := range actions {
		txins, txouts, inputs, err := action.Build(ctx)
		if err != nil {
			return nil, errors.WithDetailf(err, "invalid action %d", i)
		}

		if len(txins) != len(inputs) {
			// This would only happen from a bug in our system
			return nil, errors.Wrap(fmt.Errorf("%T returned different number of transaction and template inputs", action))
		}

		for i := range txins {
			inputs[i].Position = uint32(len(tx.Inputs))
			tplInputs = append(tplInputs, inputs[i])
			tx.Inputs = append(tx.Inputs, txins[i])
		}

		tx.Outputs = append(tx.Outputs, txouts...)
	}

	for _, input := range tplInputs {
		// Empty signature arrays should be serialized as empty arrays, not null.
		if input.SigComponents == nil {
			input.SigComponents = []*SigScriptComponent{}
		}
	}

	tpl := &Template{
		Unsigned: tx,
		Inputs:   tplInputs,
		Local:    true,
	}
	ComputeSigHashes(tpl)
	return tpl, nil
}

// ComputeSigHashes populates signature data for every input and sigscript
// component.
func ComputeSigHashes(tpl *Template) {
	sigHasher := bc.NewSigHasher(tpl.Unsigned)
	for i, in := range tpl.Inputs {
		h := sigHasher.Hash(i, bc.SigHashAll)
		for _, c := range in.SigComponents {
			c.SignatureData = h
		}
	}
}

// AssembleSignatures takes a filled in Template
// and adds the signatures to the template's unsigned transaction,
// creating a fully-signed transaction.
func AssembleSignatures(txTemplate *Template) (*bc.Tx, error) {
	msg := txTemplate.Unsigned
	for i, input := range txTemplate.Inputs {
		if msg.Inputs[input.Position] == nil {
			return nil, errors.WithDetailf(ErrBadTxInputIdx, "input %d references missing tx input %d", i, input.Position)
		}

		components := input.SigComponents

		witness := make([][]byte, 0, len(components))

		for _, c := range components {
			switch c.Type {
			case "data":
				witness = append(witness, c.Data)
			case "signature":
				added := 0
				for _, sig := range c.Signatures {
					if len(sig.Bytes) == 0 {
						continue
					}
					witness = append(witness, sig.Bytes)
					added++
					if added == c.Quorum {
						break
					}
				}
				if added < c.Quorum {
					return nil, errors.WithDetailf(ErrMissingSig, "input %d requires %d signatures, got %d", i, c.Quorum, added)
				}
			default:
				return nil, errors.WithDetailf(ErrBadSigScriptComp, "input %d unknown type %s", i, c.Type)
			}
		}
		msg.Inputs[input.Position].InputWitness = witness
	}

	return bc.NewTx(*msg), nil
}

// InputSigs takes a set of keys
// and creates a matching set of Input Signatures
// for a Template
func InputSigs(keys []*hd25519.XPub, path []uint32) (sigs []*Signature) {
	sigs = []*Signature{}
	for _, k := range keys {
		sigs = append(sigs, &Signature{
			XPub:           k.String(),
			DerivationPath: path,
		})
	}
	return sigs
}

func Sign(ctx context.Context, tpl *Template, signFn func(context.Context, *SigScriptComponent, *Signature) ([]byte, error)) error {
	ComputeSigHashes(tpl)
	// TODO(kr): come up with some scheme to verify that the
	// covered output scripts are what the client really wants.
	for i, input := range tpl.Inputs {
		if len(input.SigComponents) > 0 {
			for c, component := range input.SigComponents {
				if component.Type != "signature" {
					continue
				}
				for s, sig := range component.Signatures {
					if len(sig.Bytes) > 0 {
						continue
					}
					sigBytes, err := signFn(ctx, component, sig)
					if err != nil {
						return errors.WithDetailf(err, "computing signature for input %d, sigscript component %d, sig %d", i, c, s)
					}
					sig.Bytes = sigBytes
				}
			}
		}
	}
	return nil
}
