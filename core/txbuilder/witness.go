package txbuilder

import (
	"bytes"
	"context"

	"chain/core/pb"
	"chain/crypto/ed25519/chainkd"
	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

// SignFunc is the function passed into Sign that produces
// a signature for a given xpub, derivation path, and hash.
type SignFunc func(context.Context, chainkd.XPub, [][]byte, [32]byte) ([]byte, error)

// WitnessComponent encodes instructions for finalizing a transaction
// by populating its InputWitness fields. Each WitnessComponent object
// produces zero or more items for the InputWitness of the txinput it
// corresponds to.
type WitnessComponent interface {
	// Sign is called to add signatures. Actual signing is delegated to
	// a callback function.
	Sign(context.Context, *Template, uint32, []chainkd.XPub, SignFunc) error

	// Materialize is called to turn the component into a vector of
	// arguments for the input witness.
	Materialize(*Template, uint32, *[][]byte) error
}

// materializeWitnesses takes a filled in Template and "materializes"
// each witness component, turning it into a vector of arguments for
// the tx's input witness, creating a fully-signed transaction.
func materializeWitnesses(txTemplate *Template) error {
	tx := txTemplate.Tx
	if tx == nil {
		return errors.Wrap(ErrMissingRawTx)
	}

	if len(txTemplate.SigningInstructions) > len(tx.Inputs) {
		return errors.Wrap(ErrBadInstructionCount)
	}

	for i, sigInst := range txTemplate.SigningInstructions {
		if tx.Inputs[sigInst.Position] == nil {
			return errors.WithDetailf(ErrBadTxInputIdx, "signing instruction %d references missing tx input %d", i, sigInst.Position)
		}

		var witness [][]byte
		for j, proto := range sigInst.WitnessComponents {
			var c WitnessComponent
			switch proto.Component.(type) {
			case *pb.TxTemplate_WitnessComponent_Signature:
				c = (*signatureWitness)(proto.GetSignature())
			}
			if c == nil {
				continue
			}
			err := c.Materialize(txTemplate, sigInst.Position, &witness)
			if err != nil {
				return errors.WithDetailf(err, "error in witness component %d of input %d", j, i)
			}
		}

		tx.Inputs[sigInst.Position].SetArguments(witness)
	}

	var buf bytes.Buffer
	_, err := tx.WriteTo(&buf)
	if err != nil {
		return err
	}
	txTemplate.TxTemplate.RawTransaction = buf.Bytes()

	return nil
}

type signatureWitness pb.TxTemplate_SignatureComponent

var ErrEmptyProgram = errors.New("empty signature program")

// Sign populates sw.Sigs with as many signatures of the predicate in
// sw.Program as it can from the overlapping set of keys in sw.Keys
// and xpubs.
//
// If sw.Program is empty, it is populated with an _inferred_ predicate:
// a program committing to aspects of the current
// transaction. Specifically, the program commits to:
//  - the mintime and maxtime of the transaction (if non-zero)
//  - the outpoint and (if non-empty) reference data of the current input
//  - the assetID, amount, control program, and (if non-empty) reference data of each output.
func (sw *signatureWitness) Sign(ctx context.Context, tpl *Template, index uint32, xpubs []chainkd.XPub, signFn SignFunc) error {
	// Compute the predicate to sign. This is either a
	// txsighash program if tpl.AllowAdditionalActions is false (i.e., the tx is complete
	// and no further changes are allowed) or a program enforcing
	// constraints derived from the existing outputs and current input.
	if len(sw.Program) == 0 {
		sw.Program = buildSigProgram(tpl, tpl.SigningInstructions[index].Position)
		if len(sw.Program) == 0 {
			return ErrEmptyProgram
		}
	}
	if len(sw.Signatures) < len(sw.KeyIds) {
		// Each key in sw.KeyIds may produce a signature in sw.Signaturess. Make
		// sure there are enough slots in sw.Sigs and that we preserve any
		// sigs already present.
		newSigs := make([][]byte, len(sw.KeyIds))
		copy(newSigs, sw.Signatures)
		sw.Signatures = newSigs
	}
	var h [32]byte
	sha3pool.Sum256(h[:], sw.Program)
	for i, keyID := range sw.KeyIds {
		var xpub chainkd.XPub
		if len(xpub) != len(keyID.Xpub) {
			continue
		}
		copy(xpub[:], keyID.Xpub)

		if len(sw.Signatures[i]) > 0 {
			// Already have a signature for this key
			continue
		}
		if !contains(xpubs, xpub) {
			continue
		}
		var path [][]byte
		for _, p := range keyID.DerivationPath {
			path = append(path, p)
		}
		sigBytes, err := signFn(ctx, xpub, path, h)
		if err != nil {
			return errors.WithDetailf(err, "computing signature %d", i)
		}
		sw.Signatures[i] = sigBytes
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
	if !tpl.AllowAdditionalActions {
		h := tpl.Hash(index)
		builder := vmutil.NewBuilder()
		builder.AddData(h[:])
		builder.AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
		return builder.Program
	}
	constraints := make([]constraint, 0, 3+len(tpl.Tx.Outputs))
	constraints = append(constraints, &timeConstraint{
		minTimeMS: tpl.Tx.MinTime,
		maxTimeMS: tpl.Tx.MaxTime,
	})
	inp := tpl.Tx.Inputs[index]
	if !inp.IsIssuance() {
		constraints = append(constraints, outpointConstraint(inp.Outpoint()))
	}

	// Commitment to the tx-level refdata is conditional on it being
	// non-empty. Commitment to the input-level refdata is
	// unconditional. Rationale: no one should be able to change "my"
	// reference data; anyone should be able to set tx refdata but, once
	// set, it should be immutable.
	if len(tpl.Tx.ReferenceData) > 0 {
		constraints = append(constraints, refdataConstraint{tpl.Tx.ReferenceData, true})
	}
	constraints = append(constraints, refdataConstraint{inp.ReferenceData, false})

	for i, out := range tpl.Tx.Outputs {
		c := &payConstraint{
			Index:       i,
			AssetAmount: out.AssetAmount,
			Program:     out.ControlProgram,
		}
		if len(out.ReferenceData) > 0 {
			var h [32]byte
			sha3pool.Sum256(h[:], out.ReferenceData)
			c.RefDataHash = (*bc.Hash)(&h)
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

func (sw *signatureWitness) Materialize(tpl *Template, index uint32, args *[][]byte) error {
	// This is the value of N for the CHECKPREDICATE call. The code
	// assumes that everything already in the arg list before this call
	// to Materialize is input to the signature program, so N is
	// len(*args).
	*args = append(*args, vm.Int64Bytes(int64(len(*args))))

	var nsigs int
	for i := 0; i < len(sw.Signatures) && nsigs < int(sw.Quorum); i++ {
		if len(sw.Signatures[i]) > 0 {
			*args = append(*args, sw.Signatures[i])
			nsigs++
		}
	}
	*args = append(*args, sw.Program)
	return nil
}
