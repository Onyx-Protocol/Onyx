package tx

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func mapTx(tx *bc.TxData) (headerRef *EntryRef, entryMap map[bc.Hash]entry, err error) {
	entryMap = make(map[bc.Hash]entry)

	addEntry := func(e entry) (ref *EntryRef, err error) {
		defer func() {
			if pErr, ok := recover().(error); ok {
				err = pErr
			}
		}()
		ref = NewEntryRef(e)
		entryMap[ref.Hash] = e
		return ref, err
	}

	// Loop twice over tx.Inputs, once for spends and once for
	// issuances.  Do spends first so the entry ID of the first spend is
	// available in case an issuance needs it for its anchor.

	var firstSpendRef *EntryRef
	muxSources := make([]valueSource, len(tx.Inputs))

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*bc.SpendInput); ok {
			var spRef *EntryRef
			spRef, err = addEntry(newSpend(NewIDRef(oldSp.SpentOutputID), hashData(inp.ReferenceData), i))
			if err != nil {
				err = errors.Wrapf(err, "adding spend entry for input %d", i)
				return
			}
			muxSources[i] = valueSource{
				Ref:   spRef,
				Value: oldSp.AssetAmount,
			}
			if firstSpendRef == nil {
				firstSpendRef = spRef
			}
		}
	}

	for i, inp := range tx.Inputs {
		if oldIss, ok := inp.TypedInput.(*bc.IssuanceInput); ok {
			// Note: asset definitions, initial block ids, and issuance
			// programs are omitted here because they do not contribute to
			// the body hash of an issuance.

			var nonceRef *EntryRef

			if len(oldIss.Nonce) == 0 {
				if firstSpendRef == nil {
					err = fmt.Errorf("nonce-less issuance in transaction with no spends")
					return
				}
				nonceRef = firstSpendRef
			} else {
				var trRef *EntryRef
				trRef, err = addEntry(newTimeRange(tx.MinTime, tx.MaxTime))
				if err != nil {
					err = errors.Wrapf(err, "adding timerange entry for input %d", i)
					return
				}

				assetID := oldIss.AssetID()
				b := vmutil.NewBuilder()
				b = b.AddData(oldIss.Nonce).AddOp(vm.OP_DROP).AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)

				nonceRef, err = addEntry(newNonce(program{1, b.Program}, trRef))
				if err != nil {
					err = errors.Wrapf(err, "adding nonce entry for input %d", i)
					return
				}
			}

			val := inp.AssetAmount()

			var issRef *EntryRef
			issRef, err = addEntry(newIssuance(nonceRef, val, hashData(inp.ReferenceData), i))
			if err != nil {
				err = errors.Wrapf(err, "adding issuance entry for input %d", i)
				return
			}

			muxSources[i] = valueSource{
				Ref:   issRef,
				Value: val,
			}
		}
	}

	muxRef, err := addEntry(newMux(muxSources, program{VMVersion: 1, Code: []byte{byte(vm.OP_TRUE)}}))
	if err != nil {
		err = errors.Wrap(err, "adding mux entry")
		return
	}

	var results []*EntryRef

	for i, out := range tx.Outputs {
		s := valueSource{
			Ref:      muxRef,
			Position: uint64(i),
			Value:    out.AssetAmount,
		}

		var resultRef *EntryRef
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			resultRef, err = addEntry(newRetirement(s, hashData(out.ReferenceData), i))
			if err != nil {
				err = errors.Wrapf(err, "adding retirement entry for output %d", i)
				return
			}
		} else {
			// non-retirement
			prog := program{out.VMVersion, out.ControlProgram}
			resultRef, err = addEntry(newOutput(s, prog, hashData(out.ReferenceData), i))
			if err != nil {
				err = errors.Wrapf(err, "adding output entry for output %d", i)
				return
			}
		}

		results = append(results, resultRef)
	}

	headerRef, err = addEntry(newHeader(tx.Version, results, hashData(tx.ReferenceData), tx.MinTime, tx.MaxTime))
	if err != nil {
		err = errors.Wrap(err, "adding header entry")
		return
	}

	return headerRef, entryMap, nil
}

func mapBlockHeader(old *bc.BlockHeader) (bhID bc.Hash, bh *blockHeader) {
	bh = newBlockHeader(old.Version, old.Height, old.PreviousBlockHash, old.TimestampMS, old.TransactionsMerkleRoot, old.AssetsMerkleRoot, old.ConsensusProgram)
	bhID = entryID(bh)
	return
}

func hashData(data []byte) (h bc.Hash) {
	sha3pool.Sum256(h[:], data)
	return
}
