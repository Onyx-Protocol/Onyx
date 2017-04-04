package validation

import (
	"bytes"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

func NewBlockVMContext(block *bc.BlockEntries, prog []byte, args [][]byte) *vm.Context {
	blockHash := block.ID[:]
	return &vm.Context{
		VMVersion: 1,
		Code:      prog,
		Arguments: args,

		BlockHash:            &blockHash,
		BlockTimeMS:          &block.Body.TimestampMs,
		NextConsensusProgram: &block.Body.NextConsensusProgram,
	}
}

func NewTxVMContext(tx *bc.TxEntries, entry bc.Entry, prog *bc.Program, args [][]byte) *vm.Context {
	var (
		numResults = uint64(len(tx.Body.ResultIds))
		txData     = tx.Body.Data.Hash().Bytes()
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
		anchored := tx.Entries[e.Witness.AnchoredId.Hash()] // xxx check
		if iss, ok := anchored.(*bc.Issuance); ok {
			a1 := iss.Body.Value.AssetId.AssetID().Bytes()
			assetID = &a1
			amount = &iss.Body.Value.Amount
		}

	case *bc.Issuance:
		a1 := e.Body.Value.AssetId.AssetID().Bytes()
		assetID = &a1
		amount = &e.Body.Value.Amount
		destPos = &e.Witness.Destination.Position
		d := e.Body.Data.Hash().Bytes()
		entryData = &d
		a2 := e.Body.AnchorId.Hash().Bytes()
		anchorID = &a2

	case *bc.Spend:
		spentOutput := tx.Entries[e.Body.SpentOutputId.Hash()].(*bc.Output) // xxx check
		a1 := spentOutput.Body.Source.Value.AssetId.AssetID().Bytes()
		assetID = &a1
		amount = &spentOutput.Body.Source.Value.Amount
		destPos = &e.Witness.Destination.Position
		d := e.Body.Data.Hash().Bytes()
		entryData = &d
		s := e.Body.SpentOutputId.Hash().Bytes()
		spentOutputID = &s

	case *bc.Output:
		d := e.Body.Data.Hash().Bytes()
		entryData = &d

	case *bc.Retirement:
		d := e.Body.Data.Hash().Bytes()
		entryData = &d
	}

	var txSigHash *[]byte
	txSigHashFn := func() []byte {
		if txSigHash == nil {
			hasher := sha3pool.Get256()
			defer sha3pool.Put256(hasher)

			hasher.Write(entryID[:])
			hasher.Write(tx.ID[:])

			var hash bc.Hash
			hasher.Read(hash[:])
			hashBytes := hash.Bytes()
			txSigHash = &hashBytes
		}
		return *txSigHash
	}

	checkOutput := func(index uint64, data []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte) (bool, error) {
		checkEntry := func(e bc.Entry) (bool, error) {
			check := func(prog *bc.Program, value bc.AssetAmount, dataHash bc.Hash) bool {
				return (prog.VmVersion == vmVersion &&
					bytes.Equal(prog.Code, code) &&
					bytes.Equal(value.AssetID[:], assetID) &&
					value.Amount == amount &&
					(len(data) == 0 || bytes.Equal(dataHash[:], data)))
			}

			switch e := e.(type) {
			case *bc.Output:
				return check(e.Body.ControlProgram, e.Body.Source.Value.AssetAmount(), e.Body.Data.Hash()), nil

			case *bc.Retirement:
				return check(&bc.Program{}, e.Body.Source.Value.AssetAmount(), e.Body.Data.Hash()), nil
			}

			return false, vm.ErrContext
		}

		checkMux := func(m *bc.Mux) (bool, error) {
			if index >= uint64(len(m.Witness.Destinations)) {
				return false, errors.Wrapf(vm.ErrBadValue, "index %d >= %d", index, len(m.Witness.Destinations))
			}
			e := tx.Entries[m.Witness.Destinations[index].Ref.Hash()] // xxx check
			return checkEntry(e)
		}

		switch e := entry.(type) {
		case *bc.Mux:
			return checkMux(e)

		case *bc.Issuance:
			d := tx.Entries[e.Witness.Destination.Ref.Hash()] // xxx check
			if m, ok := d.(*bc.Mux); ok {
				return checkMux(m)
			}
			if index != 0 {
				return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
			}
			return checkEntry(d)

		case *bc.Spend:
			d := tx.Entries[e.Witness.Destination.Ref.Hash()] // xxx check
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

	result := &vm.Context{
		VMVersion: prog.VmVersion,
		Code:      prog.Code,
		Arguments: args,

		EntryID: entryID[:],

		TxVersion: &tx.Body.Version,

		TxSigHash:     txSigHashFn,
		NumResults:    &numResults,
		AssetID:       assetID,
		Amount:        amount,
		MinTimeMS:     &tx.Body.MinTimeMs,
		MaxTimeMS:     &tx.Body.MaxTimeMs,
		EntryData:     entryData,
		TxData:        &txData,
		DestPos:       destPos,
		AnchorID:      anchorID,
		SpentOutputID: spentOutputID,
		CheckOutput:   checkOutput,
	}

	return result
}
