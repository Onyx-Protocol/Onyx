package txbuilder

import (
	"context"
	"encoding/json"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

// WitnessComponent encodes instructions for finalizing a transaction
// by populating its InputWitness fields. Each WitnessComponent object
// produces zero or more items for the InputWitness of the txinput it
// corresponds to.
type WitnessComponent interface {
	// Sign is called to add signatures. Actual signing is delegated to
	// a callback function.
	Sign(context.Context, *Template, int, func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error

	// Materialize is called to turn the component into a vector of
	// arguments for the input witness.
	Materialize(*Template, int) ([][]byte, error)
}

// materializeWitnesses takes a filled in Template and "materializes"
// each witness component, turning it into a vector of arguments for
// the tx's input witness, creating a fully-signed transaction.
func materializeWitnesses(txTemplate *Template) error {
	msg := txTemplate.Transaction

	if msg == nil {
		return errors.Wrap(ErrMissingRawTx)
	}

	if len(txTemplate.Inputs) > len(msg.Inputs) {
		return errors.Wrap(ErrBadInputCount)
	}

	for i, input := range txTemplate.Inputs {
		if msg.Inputs[input.Position] == nil {
			return errors.WithDetailf(ErrBadTxInputIdx, "input %d references missing tx input %d", i, input.Position)
		}

		var witness [][]byte
		for j, c := range input.WitnessComponents {
			items, err := c.Materialize(txTemplate, i)
			if err != nil {
				return errors.WithDetailf(err, "error in witness component %d of input %d", j, i)
			}
			witness = append(witness, items...)
		}

		msg.Inputs[input.Position].InputWitness = witness
	}

	return nil
}

type DataWitness []byte

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
		// predicate in the txinput. An empty constraint list produces a
		// deferred predicate that commits to the tx sighash.
		Constraints ConstraintList `json:"constraints"`

		// Sigs is the output of Sign, where program (the output of Stage)
		// is signed by each of the keys in Keys.
		Sigs []chainjson.HexBytes `json:"signatures"`
	}

	KeyID struct {
		XPub           string   `json:"xpub"`
		DerivationPath []uint32 `json:"derivation_path"`
	}
)

func (sw *SignatureWitness) stage(tpl *Template, index int) []byte {
	if len(sw.Constraints) == 0 {
		h := tpl.Hash(index, bc.SigHashAll)
		builder := vmutil.NewBuilder()
		builder.AddData(h[:])
		builder.AddInt64(1).AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
		return builder.Program
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
	program := sw.stage(tpl, index)
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
	input := tpl.Transaction.Inputs[index]
	var multiSig []byte
	if input.IsIssuance() {
		multiSig = input.IssuanceProgram()
	} else {
		multiSig = input.ControlProgram()
	}
	pubkeys, quorum, err := vmutil.ParseP2DPMultiSigProgram(multiSig)
	if err != nil {
		return nil, errors.Wrap(err, "parsing input program script")
	}
	var sigs [][]byte
	program := sw.stage(tpl, index)
	h := sha3.Sum256(program)
	for i := 0; i < len(pubkeys) && len(sigs) < quorum; i++ {
		k := indexSig(pubkeys[i], h[:], sw.Sigs)
		if k >= 0 {
			sigs = append(sigs, sw.Sigs[k])
		}
	}
	return append(sigs, program), nil
}

func indexSig(key ed25519.PublicKey, msg []byte, sigs []chainjson.HexBytes) int {
	for i, sig := range sigs {
		if ed25519.Verify(key, msg, sig) {
			return i
		}
	}
	return -1
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
