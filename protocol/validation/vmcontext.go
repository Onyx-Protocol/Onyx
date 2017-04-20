package validation

import (
	"bytes"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

func newBlockVMContext(block *bc.Block, prog []byte, args [][]byte) *vm.Context {
	blockHash := block.ID.Bytes()
	return &vm.Context{
		VMVersion: 1,
		Code:      prog,
		Arguments: args,

		BlockHash:            &blockHash,
		BlockTimeMS:          &block.TimestampMs,
		NextConsensusProgram: &block.NextConsensusProgram,
	}
}

func NewTxVMContext(tx *bc.Tx, entry bc.Entry, prog *bc.Program, args [][]byte) *vm.Context {
	var (
		numResults = uint64(len(tx.ResultIds))
		txData     = tx.Data.Bytes()
		entryID    = bc.EntryID(entry) // TODO(bobg): pass this in, don't recompute it

		assetID       *[]byte
		amount        *uint64
		entryData     *[]byte
		destPos       *uint64
		anchorID      *[]byte
		spentOutputID *[]byte
	)

	switch e := entry.(type) {
	case *bc.Nonce:
		anchored := tx.Entries[*e.WitnessAnchoredId]
		if iss, ok := anchored.(*bc.Issuance); ok {
			a1 := iss.Value.AssetId.Bytes()
			assetID = &a1
			amount = &iss.Value.Amount
		}

	case *bc.Issuance:
		a1 := e.Value.AssetId.Bytes()
		assetID = &a1
		amount = &e.Value.Amount
		destPos = &e.WitnessDestination.Position
		d := e.Data.Bytes()
		entryData = &d
		a2 := e.AnchorId.Bytes()
		anchorID = &a2

	case *bc.Spend:
		spentOutput := tx.Entries[*e.SpentOutputId].(*bc.Output)
		a1 := spentOutput.Source.Value.AssetId.Bytes()
		assetID = &a1
		amount = &spentOutput.Source.Value.Amount
		destPos = &e.WitnessDestination.Position
		d := e.Data.Bytes()
		entryData = &d
		s := e.SpentOutputId.Bytes()
		spentOutputID = &s

	case *bc.Output:
		d := e.Data.Bytes()
		entryData = &d

	case *bc.Retirement:
		d := e.Data.Bytes()
		entryData = &d
	}

	var txSigHash *[]byte
	txSigHashFn := func() []byte {
		if txSigHash == nil {
			hasher := sha3pool.Get256()
			defer sha3pool.Put256(hasher)

			entryID.WriteTo(hasher)
			tx.ID.WriteTo(hasher)

			var hash bc.Hash
			hash.ReadFrom(hasher)
			hashBytes := hash.Bytes()
			txSigHash = &hashBytes
		}
		return *txSigHash
	}

	ec := &entryContext{
		entry:   entry,
		entries: tx.Entries,
	}

	result := &vm.Context{
		VMVersion: prog.VmVersion,
		Code:      prog.Code,
		Arguments: args,

		EntryID: entryID.Bytes(),

		TxVersion: &tx.Version,

		TxSigHash:     txSigHashFn,
		NumResults:    &numResults,
		AssetID:       assetID,
		Amount:        amount,
		MinTimeMS:     &tx.MinTimeMs,
		MaxTimeMS:     &tx.MaxTimeMs,
		EntryData:     entryData,
		TxData:        &txData,
		DestPos:       destPos,
		AnchorID:      anchorID,
		SpentOutputID: spentOutputID,
		CheckOutput:   ec.checkOutput,
	}

	return result
}

type entryContext struct {
	entry   bc.Entry
	entries map[bc.Hash]bc.Entry
}

func (ec *entryContext) checkOutput(index uint64, data []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte, expansion bool) (bool, error) {
	checkEntry := func(e bc.Entry) (bool, error) {
		check := func(prog *bc.Program, value *bc.AssetAmount, dataHash *bc.Hash) bool {
			return (prog.VmVersion == vmVersion &&
				bytes.Equal(prog.Code, code) &&
				bytes.Equal(value.AssetId.Bytes(), assetID) &&
				value.Amount == amount &&
				(len(data) == 0 || bytes.Equal(dataHash.Bytes(), data)))
		}

		switch e := e.(type) {
		case *bc.Output:
			return check(e.ControlProgram, e.Source.Value, e.Data), nil

		case *bc.Retirement:
			var prog bc.Program
			if expansion {
				// The spec requires prog.Code to be the empty string only
				// when !expansion. When expansion is true, we prepopulate
				// prog.Code to give check() a freebie match.
				//
				// (The spec always requires prog.VmVersion to be zero.)
				prog.Code = code
			}
			return check(&prog, e.Source.Value, e.Data), nil
		}

		return false, vm.ErrContext
	}

	checkMux := func(m *bc.Mux) (bool, error) {
		if index >= uint64(len(m.WitnessDestinations)) {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= %d", index, len(m.WitnessDestinations))
		}
		eID := m.WitnessDestinations[index].Ref
		e, ok := ec.entries[*eID]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for mux destination %d, id %x, not found", index, eID.Bytes())
		}
		return checkEntry(e)
	}

	switch e := ec.entry.(type) {
	case *bc.Mux:
		return checkMux(e)

	case *bc.Issuance:
		d, ok := ec.entries[*e.WitnessDestination.Ref]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for issuance destination %x not found", e.WitnessDestination.Ref.Bytes())
		}
		if m, ok := d.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(d)

	case *bc.Spend:
		d, ok := ec.entries[*e.WitnessDestination.Ref]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for spend destination %x not found", e.WitnessDestination.Ref.Bytes())
		}
		if m, ok := d.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(d)
	}

	return false, vm.ErrContext
}
