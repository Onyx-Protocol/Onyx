package tx

import (
	"fmt"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func mapTx(tx *bc.TxData) (headerID entryRef, hdr *header, entryMap map[entryRef]entry, err error) {
	var refdataID entryRef

	entryMap = make(map[entryRef]entry)

	addEntry := func(e entry) (id entryRef, entry entry, err error) {
		id, err = entryID(e)
		if err != nil {
			err = errors.Wrapf(err, "computing entryID for %s entry", e.Type())
			return
		}
		entryMap[id] = e
		return id, e, nil
	}

	if len(tx.ReferenceData) > 0 {
		refdataID, _, err = addEntry(newData(hashData(tx.ReferenceData)))
		if err != nil {
			err = errors.Wrap(err, "adding refdata entry")
			return
		}
	}

	// Loop twice over tx.Inputs, once for spends and once for
	// issuances.  Do spends first so the entry ID of the first spend is
	// available in case an issuance needs it for its anchor.

	var firstSpendID *entryRef
	muxSources := make([]valueSource, len(tx.Inputs))

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*bc.SpendInput); ok {
			var inpRefdataID entryRef
			if len(inp.ReferenceData) != 0 {
				inpRefdataID, _, err = addEntry(newData(hashData(inp.ReferenceData)))
				if err != nil {
					return
				}
			}
			var spID entryRef
			spID, _, err = addEntry(newSpend(entryRef(oldSp.SpentOutputID.Hash), inpRefdataID, i))
			if err != nil {
				err = errors.Wrapf(err, "adding spend entry for input %d", i)
				return
			}
			muxSources[i] = valueSource{
				Ref:   spID,
				Value: oldSp.AssetAmount,
			}
			if firstSpendID == nil {
				firstSpendID = &spID
			}
		}
	}

	for i, inp := range tx.Inputs {
		if oldIss, ok := inp.TypedInput.(*bc.IssuanceInput); ok {
			var inpRefdataID entryRef
			if len(inp.ReferenceData) != 0 {
				inpRefdataID, _, err = addEntry(newData(hashData(inp.ReferenceData)))
				if err != nil {
					err = errors.Wrapf(err, "adding input refdata entry for input %d", i)
					return
				}
			}

			// Note: asset definitions, initial block ids, and issuance
			// programs are omitted here because they do not contribute to
			// the body hash of an issuance.

			var nonceHash entryRef

			if len(oldIss.Nonce) == 0 {
				if firstSpendID == nil {
					err = fmt.Errorf("nonce-less issuance in transaction with no spends")
					return
				}
				nonceHash = *firstSpendID
			} else {
				var trID entryRef
				trID, _, err = addEntry(newTimeRange(tx.MinTime, tx.MaxTime))
				if err != nil {
					err = errors.Wrapf(err, "adding timerange entry for input %d", i)
					return
				}

				assetID := oldIss.AssetID()
				b := vmutil.NewBuilder()
				b = b.AddData(oldIss.Nonce).AddOp(vm.OP_DROP).AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)

				nonceHash, _, err = addEntry(newNonce(program{1, b.Program}, trID))
				if err != nil {
					err = errors.Wrapf(err, "adding nonce entry for input %d", i)
					return
				}
			}

			val := inp.AssetAmount()

			var issID entryRef
			issID, _, err = addEntry(newIssuance(nonceHash, val, inpRefdataID, i))
			if err != nil {
				err = errors.Wrapf(err, "adding issuance entry for input %d", i)
				return
			}

			muxSources[i] = valueSource{
				Ref:   issID,
				Value: val,
			}
		}
	}

	muxID, _, err := addEntry(newMux(muxSources))
	if err != nil {
		err = errors.Wrap(err, "adding mux entry")
		return
	}

	var results []entryRef

	for i, out := range tx.Outputs {
		s := valueSource{
			Ref:      muxID,
			Position: uint64(i),
			Value:    out.AssetAmount,
		}

		var outRefdataID entryRef
		if len(out.ReferenceData) > 0 {
			outRefdataID, _, err = addEntry(newData(hashData(out.ReferenceData)))
			if err != nil {
				err = errors.Wrapf(err, "adding refdata entry for output %d", i)
				return
			}
		}

		var resultID entryRef
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			resultID, _, err = addEntry(newRetirement(s, outRefdataID, i))
			if err != nil {
				err = errors.Wrapf(err, "adding retirement entry for output %d", i)
				return
			}
		} else {
			// non-retirement
			prog := program{out.VMVersion, out.ControlProgram}
			resultID, _, err = addEntry(newOutput(s, prog, outRefdataID, i))
			if err != nil {
				err = errors.Wrapf(err, "adding output entry for output %d", i)
				return
			}
		}

		results = append(results, resultID)
	}

	var h entry
	headerID, h, err = addEntry(newHeader(tx.Version, results, refdataID, tx.MinTime, tx.MaxTime))
	if err != nil {
		err = errors.Wrap(err, "adding header entry")
		return
	}

	return headerID, h.(*header), entryMap, nil
}
