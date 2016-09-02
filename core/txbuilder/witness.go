package txbuilder

import (
	"context"
	"encoding/json"

	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

// WitnessComponent encodes instructions for finalizing a transaction
// by populating its InputWitness fields. Each WitnessComponent object
// produces zero or more items for the InputWitness of the txinput it
// corresponds to.
type WitnessComponent interface {
	// Stage is called on the component after all the inputs of a tx
	// template are present (e.g., to add the tx sighash).
	Stage(*Template, int)

	// Sign is called to add signatures. Actual signing is delegated to
	// a callback function.
	Sign(context.Context, func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error

	// Materialize is called to turn the component into a vector of
	// arguments for the input witness.
	Materialize() ([][]byte, error)
}

// StageWitnesses "stages" each witness component by e.g. populating
// signature data with the tx sighash.
func StageWitnesses(tpl *Template) {
	for i, in := range tpl.Inputs {
		for _, c := range in.WitnessComponents {
			c.Stage(tpl, i)
		}
	}
}

// MaterializeWitnesses takes a filled in Template and "materializes"
// each witness component, turning it into a vector of arguments for
// the tx's input witness, creating a fully-signed transaction.
func MaterializeWitnesses(txTemplate *Template) (*bc.Tx, error) {
	msg := txTemplate.Unsigned
	for i, input := range txTemplate.Inputs {
		if msg.Inputs[input.Position] == nil {
			return nil, errors.WithDetailf(ErrBadTxInputIdx, "input %d references missing tx input %d", i, input.Position)
		}

		var witness [][]byte
		for j, c := range input.WitnessComponents {
			items, err := c.Materialize()
			if err != nil {
				return nil, errors.WithDetailf(err, "error in witness component %d of input %d", j, i)
			}
			witness = append(witness, items...)
		}

		msg.Inputs[input.Position].InputWitness = witness
	}

	return bc.NewTx(*msg), nil
}

type DataWitness []byte

func (_ DataWitness) Stage(_ *Template, _ int) {}
func (_ DataWitness) Sign(_ context.Context, _ func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error {
	return nil
}

func (d DataWitness) Materialize() ([][]byte, error) {
	return [][]byte{d}, nil
}

func (d DataWitness) MarshalJSON() ([]byte, error) {
	obj := struct {
		Type string             `json:"type"`
		Data chainjson.HexBytes `json:"data"`
	}{
		Type: "data",
		Data: chainjson.HexBytes(d),
	}
	return json.Marshal(obj)
}

type (
	SignatureWitness struct {
		// The number of signatures required
		Quorum int `json:"quorum"`

		// The data to be signed
		SignatureData bc.Hash `json:"signature_data"`

		// Identities of the keys to sign with (and, upon finalizing, the
		// signature bytes themselves)
		Signatures []*Signature `json:"signatures"`
	}

	Signature struct {
		XPub           string             `json:"xpub"`
		DerivationPath []uint32           `json:"derivation_path"`
		Bytes          chainjson.HexBytes `json:"signature"`
	}
)

func (sw *SignatureWitness) Stage(tpl *Template, index int) {
	sw.SignatureData = tpl.Hash(index, bc.SigHashAll)
}

func (sw *SignatureWitness) Sign(ctx context.Context, signFn func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error {
	for i, sig := range sw.Signatures {
		if len(sig.Bytes) > 0 {
			continue
		}
		sigBytes, err := signFn(ctx, sig.XPub, sig.DerivationPath, sw.SignatureData)
		if err != nil {
			return errors.WithDetailf(err, "computing signature %d", i)
		}
		sig.Bytes = sigBytes
	}
	return nil
}

func (sw SignatureWitness) Materialize() ([][]byte, error) {
	added := 0
	var result [][]byte
	for _, s := range sw.Signatures {
		if len(s.Bytes) == 0 {
			continue
		}
		result = append(result, s.Bytes)
		added++
		if added >= sw.Quorum {
			break
		}
	}
	if added < sw.Quorum {
		return nil, errors.WithDetailf(ErrMissingSig, "requires %d signature(s), got %d", sw.Quorum, added)
	}
	return result, nil
}

func (sw SignatureWitness) MarshalJSON() ([]byte, error) {
	obj := struct {
		Type          string       `json:"type"`
		Quorum        int          `json:"quorum"`
		SignatureData bc.Hash      `json:"signature_data"`
		Signatures    []*Signature `json:"signatures"`
	}{
		Type:          "signature",
		Quorum:        sw.Quorum,
		SignatureData: sw.SignatureData,
		Signatures:    sw.Signatures,
	}
	return json.Marshal(obj)
}

func (inp *Input) AddWitnessData(data []byte) {
	inp.WitnessComponents = append(inp.WitnessComponents, DataWitness(data))
}

func (inp *Input) AddWitnessSigs(sigs []*Signature, quorum int, sigData *bc.Hash) {
	sw := &SignatureWitness{
		Quorum:     quorum,
		Signatures: sigs,
	}
	if sigData != nil {
		sw.SignatureData = *sigData
	}
	inp.WitnessComponents = append(inp.WitnessComponents, sw)
}
