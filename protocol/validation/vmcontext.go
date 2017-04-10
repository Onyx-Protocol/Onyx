package validation

import (
	"bytes"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

func newBlockVMContext(block *bc.BlockEntries, prog []byte, args [][]byte) *vm.Context {
	blockHash := block.ID.Bytes()
	return &vm.Context{
		VMVersion: 1,
		Code:      prog,
		Arguments: args,

		BlockHash:            &blockHash,
		BlockTimeMS:          &block.Body.TimestampMS,
		NextConsensusProgram: &block.Body.NextConsensusProgram,
	}
}

func NewTxVMContext(tx *bc.TxEntries, entry bc.Entry, prog bc.Program, args [][]byte) *vm.Context {
	var (
		numResults = uint64(len(tx.Results))
		txData     = tx.Body.Data.Bytes()
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
		if iss, ok := e.Anchored.(*bc.Issuance); ok {
			a1 := iss.Body.Value.AssetID[:]
			assetID = &a1
			amount = &iss.Body.Value.Amount
		}

	case *bc.Issuance:
		a1 := e.Body.Value.AssetID[:]
		assetID = &a1
		amount = &e.Body.Value.Amount
		destPos = &e.Witness.Destination.Position
		d := e.Body.Data.Bytes()
		entryData = &d
		a2 := e.Body.AnchorID.Bytes()
		anchorID = &a2

	case *bc.Spend:
		a1 := e.SpentOutput.Body.Source.Value.AssetID[:]
		assetID = &a1
		amount = &e.SpentOutput.Body.Source.Value.Amount
		destPos = &e.Witness.Destination.Position
		d := e.Body.Data.Bytes()
		entryData = &d
		s := e.Body.SpentOutputID.Bytes()
		spentOutputID = &s

	case *bc.Output:
		d := e.Body.Data.Bytes()
		entryData = &d

	case *bc.Retirement:
		d := e.Body.Data.Bytes()
		entryData = &d
	}

	var txSigHash *[]byte
	txSigHashFn := func() []byte {
		if txSigHash == nil {
			hasher := sha3pool.Get256()
			defer sha3pool.Put256(hasher)

			hasher.Write(entryID.Bytes())
			hasher.Write(tx.ID.Bytes())

			var hash bc.Hash
			hash.ReadFrom(hasher)
			hashBytes := hash.Bytes()
			txSigHash = &hashBytes
		}
		return *txSigHash
	}

	result := &vm.Context{
		VMVersion: prog.VMVersion,
		Code:      prog.Code,
		Arguments: args,

		EntryID: entryID.Bytes(),

		TxVersion: &tx.Body.Version,

		TxSigHash:     txSigHashFn,
		NumResults:    &numResults,
		AssetID:       assetID,
		Amount:        amount,
		MinTimeMS:     &tx.Body.MinTimeMS,
		MaxTimeMS:     &tx.Body.MaxTimeMS,
		EntryData:     entryData,
		TxData:        &txData,
		DestPos:       destPos,
		AnchorID:      anchorID,
		SpentOutputID: spentOutputID,
		CheckOutput:   (&entryContext{entry: entry}).checkOutput,
	}

	return result
}

type entryContext struct {
	entry bc.Entry
}

func (tc *entryContext) checkOutput(index uint64, data []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte) (bool, error) {
	checkEntry := func(e bc.Entry) (bool, error) {
		check := func(prog bc.Program, value bc.AssetAmount, dataHash bc.Hash) bool {
			return (prog.VMVersion == vmVersion &&
				bytes.Equal(prog.Code, code) &&
				bytes.Equal(value.AssetID[:], assetID) &&
				value.Amount == amount &&
				(len(data) == 0 || bytes.Equal(dataHash.Bytes(), data)))
		}

		switch e := e.(type) {
		case *bc.Output:
			return check(e.Body.ControlProgram, e.Body.Source.Value, e.Body.Data), nil

		case *bc.Retirement:
			return check(bc.Program{}, e.Body.Source.Value, e.Body.Data), nil
		}

		return false, vm.ErrContext
	}

	checkMux := func(m *bc.Mux) (bool, error) {
		if index >= uint64(len(m.Witness.Destinations)) {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= %d", index, len(m.Witness.Destinations))
		}
		return checkEntry(m.Witness.Destinations[index].Entry)
	}

	switch e := tc.entry.(type) {
	case *bc.Mux:
		return checkMux(e)

	case *bc.Issuance:
		if m, ok := e.Witness.Destination.Entry.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(e.Witness.Destination.Entry)

	case *bc.Spend:
		if m, ok := e.Witness.Destination.Entry.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(e.Witness.Destination.Entry)
	}

	return false, vm.ErrContext
}
