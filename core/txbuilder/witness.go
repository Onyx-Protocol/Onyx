package txbuilder

import (
	"context"
	"encoding/json"

	"golang.org/x/crypto/sha3"

	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

// WitnessComponent encodes instructions for finalizing a transaction
// by populating its InputWitness fields. Each WitnessComponent object
// produces zero or more items for the InputWitness of the txinput it
// corresponds to.
type WitnessComponent interface {
	// Stage is called on the component after all the inputs of a tx
	// template are present (e.g., to add the tx sighash). It produces a
	// p2dp predicate.
	Stage(*Template, int) []byte

	// Sign is called to add signatures. Actual signing is delegated to
	// a callback function.
	Sign(context.Context, *Template, int, func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error

	// Materialize is called to turn the component into a vector of
	// arguments for the input witness.
	Materialize(*Template, int) ([][]byte, error)
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
			items, err := c.Materialize(txTemplate, i)
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

func (_ DataWitness) Stage(_ *Template, _ int) []byte { return nil }
func (_ DataWitness) Sign(_ context.Context, _ *Template, _ int, _ func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error {
	return nil
}

func (d DataWitness) Materialize(_ *Template, _ int) ([][]byte, error) {
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
		// Quorum is the number of signatures required.
		Quorum int `json:"quorum"`

		// Keys are the identities of the keys to sign with.
		Keys []KeyID `json:"keys"`

		// Constraints is a list of constraints to express in the deferred
		// predicate in the txinput.
		Constraints []Constraint `json:"constraints"`

		// Sigs is the output of Sign, where program (the output of Stage)
		// is signed by each of the keys in Keys.
		Sigs []chainjson.HexBytes `json:"signatures"`
	}

	KeyID struct {
		XPub           string   `json:"xpub"`
		DerivationPath []uint32 `json:"derivation_path"`
	}
)

func (sw *SignatureWitness) Stage(tpl *Template, index int) []byte {
	if len(sw.Constraints) == 0 {
		// When in doubt, commit to the hash of the current tx
		// TODO(bobg): When we add other Constraint types, require callers
		// to specify this explicitly rather than as a default.
		h := tpl.Hash(index, bc.SigHashAll)
		sw.Constraints = []Constraint{TxHashConstraint(h)}
	}
	var program []byte
	for i, c := range sw.Constraints {
		program = append(program, c.Code()...)
		if i < len(sw.Constraints)-1 { // leave the final bool on top of the stack
			program = append(program, byte(vm.OP_VERIFY))
		}
	}
	return program
}

func (sw *SignatureWitness) Sign(ctx context.Context, tpl *Template, index int, signFn func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error {
	if len(sw.Sigs) < len(sw.Keys) {
		// Each key in sw.Keys will produce a signature in sw.Sigs. Make
		// sure there are enough slots in sw.Sigs and that we preserve any
		// sigs already present.
		newSigs := make([]chainjson.HexBytes, len(sw.Keys))
		copy(newSigs, sw.Sigs)
		sw.Sigs = newSigs
	}
	program := sw.Stage(tpl, index)
	h := sha3.Sum256(program)
	for i, keyID := range sw.Keys {
		if len(sw.Sigs[i]) > 0 {
			// Already have a signature for this key
			continue
		}
		sigBytes, err := signFn(ctx, keyID.XPub, keyID.DerivationPath, h)
		if err != nil {
			return errors.WithDetailf(err, "computing signature %d", i)
		}
		sw.Sigs[i] = sigBytes
	}
	return nil
}

func (sw SignatureWitness) Materialize(tpl *Template, index int) ([][]byte, error) {
	added := 0
	result := make([][]byte, 0, 1+len(sw.Keys))
	for _, s := range sw.Sigs {
		if len(s) == 0 {
			continue
		}
		result = append(result, s)
		added++
		if added >= sw.Quorum {
			break
		}
	}
	if added < sw.Quorum {
		return nil, errors.WithDetailf(ErrMissingSig, "requires %d signature(s), got %d", sw.Quorum, added)
	}
	program := sw.Stage(tpl, index)
	result = append(result, program)
	return result, nil
}

func (sw SignatureWitness) MarshalJSON() ([]byte, error) {
	obj := struct {
		Type        string               `json:"type"`
		Quorum      int                  `json:"quorum"`
		Keys        []KeyID              `json:"keys"`
		Constraints []Constraint         `json:"constraints"`
		Sigs        []chainjson.HexBytes `json:"signatures"`
	}{
		Type:        "signature",
		Quorum:      sw.Quorum,
		Keys:        sw.Keys,
		Constraints: sw.Constraints,
		Sigs:        sw.Sigs,
	}
	return json.Marshal(obj)
}

func (sw *SignatureWitness) UnmarshalJSON(b []byte) error {
	var pre struct {
		Quorum      int     `json:"quorum"`
		Keys        []KeyID `json:"keys"`
		Constraints []json.RawMessage
		Sigs        []chainjson.HexBytes `json:"signatures"`
	}
	err := json.Unmarshal(b, &pre)
	if err != nil {
		return err
	}
	sw.Quorum = pre.Quorum
	sw.Keys = pre.Keys
	sw.Sigs = pre.Sigs
	for i, c := range pre.Constraints {
		var t struct {
			Type string `json:"type"`
		}
		err = json.Unmarshal(c, &t)
		if err != nil {
			return err
		}
		var constraint Constraint
		switch t.Type {
		case "transaction_id":
			var txhash struct {
				Hash bc.Hash `json:"transaction_id"`
			}
			err = json.Unmarshal(c, &txhash)
			if err != nil {
				return err
			}
			constraint = TxHashConstraint(txhash.Hash)
		default:
			return errors.WithDetailf(ErrBadConstraint, "constraint %d has unknown type '%s'", i, t.Type)
		}
		sw.Constraints = append(sw.Constraints, constraint)
	}
	return nil
}

func (inp *Input) AddWitnessData(data []byte) {
	inp.WitnessComponents = append(inp.WitnessComponents, DataWitness(data))
}

func (inp *Input) AddWitnessKeys(keys []KeyID, quorum int, constraints []Constraint) {
	sw := &SignatureWitness{
		Quorum:      quorum,
		Keys:        keys,
		Constraints: constraints,
	}
	inp.WitnessComponents = append(inp.WitnessComponents, sw)
}
