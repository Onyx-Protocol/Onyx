package txbuilder

import (
	"bytes"
	"context"
	"encoding/json"

	"chain/crypto/ed25519/chainkd"
	"chain/crypto/sha3pool"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vm/vmutil"
)

// SignFunc is the function passed into Sign that produces
// a signature for a given xpub, derivation path, and hash.
type SignFunc func(context.Context, chainkd.XPub, [][]byte, [32]byte) ([]byte, error)

// materializeWitnesses takes a filled in Template and "materializes"
// each witness component, turning it into a vector of arguments for
// the tx's input witness, creating a fully-signed transaction.
func materializeWitnesses(txTemplate *Template) error {
	msg := txTemplate.Transaction

	if msg == nil {
		return errors.Wrap(ErrMissingRawTx)
	}

	if len(txTemplate.SigningInstructions) > len(msg.Inputs) {
		return errors.Wrap(ErrBadInstructionCount)
	}

	for i, sigInst := range txTemplate.SigningInstructions {
		if msg.Inputs[sigInst.Position] == nil {
			return errors.WithDetailf(ErrBadTxInputIdx, "signing instruction %d references missing tx input %d", i, sigInst.Position)
		}

		var witness [][]byte
		for j, sw := range sigInst.SignatureWitnesses {
			err := sw.materialize(txTemplate, sigInst.Position, &witness)
			if err != nil {
				return errors.WithDetailf(err, "error in witness component %d of input %d", j, i)
			}
		}

		msg.SetInputArguments(sigInst.Position, witness)
	}

	return nil
}

type (
	signatureWitness struct {
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
func (sw *signatureWitness) sign(ctx context.Context, tpl *Template, index uint32, xpubs []chainkd.XPub, signFn SignFunc) error {
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
		if !contains(xpubs, keyID.XPub) {
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

func contains(list []chainkd.XPub, key chainkd.XPub) bool {
	for _, k := range list {
		if bytes.Equal(k[:], key[:]) {
			return true
		}
	}
	return false
}

func buildSigProgram(tpl *Template, index uint32) []byte {
	if !tpl.AllowAdditional {
		h := tpl.Hash(index)
		builder := vmutil.NewBuilder()
		builder.AddData(h.Bytes())
		builder.AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
		prog, _ := builder.Build() // error is impossible
		return prog
	}
	constraints := make([]constraint, 0, 3+len(tpl.Transaction.Outputs))
	constraints = append(constraints, &timeConstraint{
		minTimeMS: tpl.Transaction.MinTime,
		maxTimeMS: tpl.Transaction.MaxTime,
	})
	id := tpl.Transaction.Tx.InputIDs[index]
	if sp, err := tpl.Transaction.Tx.Spend(id); err == nil {
		constraints = append(constraints, outputIDConstraint(*sp.SpentOutputId))
	}

	// Commitment to the tx-level refdata is conditional on it being
	// non-empty. Commitment to the input-level refdata is
	// unconditional. Rationale: no one should be able to change "my"
	// reference data; anyone should be able to set tx refdata but, once
	// set, it should be immutable.
	if len(tpl.Transaction.ReferenceData) > 0 {
		constraints = append(constraints, refdataConstraint{tpl.Transaction.ReferenceData, true})
	}
	constraints = append(constraints, refdataConstraint{tpl.Transaction.Inputs[index].ReferenceData, false})

	for i, out := range tpl.Transaction.Outputs {
		c := &payConstraint{
			Index:       i,
			AssetAmount: out.AssetAmount,
			Program:     out.ControlProgram,
		}
		if len(out.ReferenceData) > 0 {
			var b32 [32]byte
			sha3pool.Sum256(b32[:], out.ReferenceData)
			h := bc.NewHash(b32)
			c.RefDataHash = &h
		}
		constraints = append(constraints, c)
	}
	var program []byte
	for i, c := range constraints {
		program = append(program, c.code()...)
		if i < len(constraints)-1 { // leave the final bool on top of the stack
			program = append(program, byte(vm.OP_VERIFY))
		}
	}
	return program
}

func (sw signatureWitness) materialize(tpl *Template, index uint32, args *[][]byte) error {
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

func (sw signatureWitness) MarshalJSON() ([]byte, error) {
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

// AddWitnessKeys adds a signatureWitness with the given quorum and
// list of keys derived by applying the derivation path to each of the
// xpubs.
func (si *SigningInstruction) AddWitnessKeys(xpubs []chainkd.XPub, path [][]byte, quorum int) {
	hexPath := make([]chainjson.HexBytes, 0, len(path))
	for _, p := range path {
		hexPath = append(hexPath, p)
	}

	keyIDs := make([]keyID, 0, len(xpubs))
	for _, xpub := range xpubs {
		keyIDs = append(keyIDs, keyID{xpub, hexPath})
	}

	sw := &signatureWitness{
		Quorum: quorum,
		Keys:   keyIDs,
	}
	si.SignatureWitnesses = append(si.SignatureWitnesses, sw)
}
