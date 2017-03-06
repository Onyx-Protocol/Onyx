package bc

import (
	"bytes"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/vm"
)

func NewBlockVMContext(block *BlockEntries, prog []byte, args [][]byte) *vm.Context {
	blockHash := block.ID[:]
	return &vm.Context{
		VMVersion: 1,
		Code:      prog,
		Arguments: args,

		BlockHash:            &blockHash,
		BlockTimeMS:          &block.Body.TimestampMS,
		NextConsensusProgram: &block.Body.NextConsensusProgram,
	}
}

func NewTxVMContext(tx *TxEntries, entry Entry, prog Program, args [][]byte) *vm.Context {
	var (
		numResults = uint64(len(tx.Results))
		txData     = tx.Body.Data[:]
		entryID    = EntryID(entry) // TODO(bobg): pass this in, don't recompute it

		assetID       *[]byte
		amount        *uint64
		entryData     *[]byte
		destPos       *uint64
		anchorID      *[]byte
		spentOutputID *[]byte
	)

	switch e := entry.(type) {
	case *Nonce:
		if iss, ok := e.Anchored.(*Issuance); ok {
			a1 := iss.Body.Value.AssetID[:]
			assetID = &a1
			amount = &iss.Body.Value.Amount
		}

	case *Issuance:
		a1 := e.Body.Value.AssetID[:]
		assetID = &a1
		amount = &e.Body.Value.Amount
		destPos = &e.Witness.Destination.Position
		d := e.Body.Data[:]
		entryData = &d
		a2 := e.Body.AnchorID[:]
		anchorID = &a2

	case *Spend:
		a1 := e.SpentOutput.Body.Source.Value.AssetID[:]
		assetID = &a1
		amount = &e.SpentOutput.Body.Source.Value.Amount
		destPos = &e.Witness.Destination.Position
		d := e.Body.Data[:]
		entryData = &d
		s := e.Body.SpentOutputID[:]
		spentOutputID = &s

	case *Output:
		d := e.Body.Data[:]
		entryData = &d

	case *Retirement:
		d := e.Body.Data[:]
		entryData = &d
	}

	var txSigHash *[]byte
	txSigHashFn := func() []byte {
		if txSigHash == nil {
			hasher := sha3pool.Get256()
			defer sha3pool.Put256(hasher)

			hasher.Write(entryID[:])
			hasher.Write(tx.ID[:])

			var hash Hash
			hasher.Read(hash[:])
			hashBytes := hash.Bytes()
			txSigHash = &hashBytes
		}
		return *txSigHash
	}

	checkOutput := func(index uint64, data []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte) (bool, error) {
		checkEntry := func(e Entry) (bool, error) {
			check := func(prog Program, value AssetAmount, dataHash Hash) bool {
				return (prog.VMVersion == vmVersion &&
					bytes.Equal(prog.Code, code) &&
					bytes.Equal(value.AssetID[:], assetID) &&
					value.Amount == amount &&
					(len(data) == 0 || bytes.Equal(dataHash[:], data)))
			}

			switch e := e.(type) {
			case *Output:
				return check(e.Body.ControlProgram, e.Body.Source.Value, e.Body.Data), nil

			case *Retirement:
				return check(Program{}, e.Body.Source.Value, e.Body.Data), nil
			}

			return false, vm.ErrContext
		}

		checkMux := func(m *Mux) (bool, error) {
			if index >= uint64(len(m.Witness.Destinations)) {
				return false, errors.Wrapf(vm.ErrBadValue, "index %d >= %d", index, len(m.Witness.Destinations))
			}
			return checkEntry(m.Witness.Destinations[index].Entry)
		}

		switch e := entry.(type) {
		case *Mux:
			return checkMux(e)

		case *Issuance:
			if m, ok := e.Witness.Destination.Entry.(*Mux); ok {
				return checkMux(m)
			}
			if index != 0 {
				return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
			}
			return checkEntry(e.Witness.Destination.Entry)

		case *Spend:
			if m, ok := e.Witness.Destination.Entry.(*Mux); ok {
				return checkMux(m)
			}
			if index != 0 {
				return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
			}
			return checkEntry(e.Witness.Destination.Entry)
		}

		return false, vm.ErrContext
	}

	result := &vm.Context{
		VMVersion: prog.VMVersion,
		Code:      prog.Code,
		Arguments: args,

		EntryID: entryID[:],

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
		CheckOutput:   checkOutput,
	}

	return result
}
