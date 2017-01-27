package tx

import (
	"fmt"

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
			return
		}
		entryMap[id] = e
		return id, e, nil
	}

	if len(tx.ReferenceData) > 0 {
		refdataID, _, err = addEntry(newData(hashData(tx.ReferenceData)))
		if err != nil {
			return
		}
	}

	// Loop twice over tx.Inputs, once for spends and once for
	// issuances.  Do spends first so the entry ID of the first spend is
	// available in case an issuance needs it for its anchor.

	var firstSpendID *entryRef
	muxSources := make([]valueSource, len(tx.Inputs))

	maybeAddInputRefdata := func(inp *bc.TxInput) (ref entryRef, err error) {
		if len(inp.ReferenceData) == 0 {
			return
		}
		ref, _, err = addEntry(newData(hashData(inp.ReferenceData)))
		return
	}

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*bc.SpendInput); ok {
			var inpRefdataID entryRef
			inpRefdataID, err = maybeAddInputRefdata(inp)
			if err != nil {
				return
			}
			var spID entryRef
			spID, _, err = addEntry(newSpend(entryRef(oldSp.SpentOutputID.Hash), inpRefdataID))
			if err != nil {
				return
			}
			muxSources[i] = valueSource{
				Ref:      spID,
				Position: uint64(i),
				Value:    oldSp.AssetAmount,
			}
			if firstSpendID == nil {
				firstSpendID = &spID
			}
		}
	}

	for i, inp := range tx.Inputs {
		if oldIss, ok := inp.TypedInput.(*bc.IssuanceInput); ok {
			var inpRefdataID entryRef
			inpRefdataID, err = maybeAddInputRefdata(inp)
			if err != nil {
				return
			}
			// xxx asset definitions, initial block ids, and issuance
			// programs are omitted here because they do not contribute to
			// the body hash of an issuance.

			var anchorHash entryRef

			if len(oldIss.Nonce) == 0 {
				if firstSpendID == nil {
					err = fmt.Errorf("nonce-less issuance in transaction with no spends")
					return
				}
				anchorHash = *firstSpendID
			} else {
				var trID entryRef
				trID, _, err = addEntry(newTimeRange(tx.MinTime, tx.MaxTime))
				if err != nil {
					return
				}

				assetID := oldIss.AssetID()
				b := vmutil.NewBuilder()
				b = b.AddData(oldIss.Nonce).AddOp(vm.OP_DROP).AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)

				anchorHash, _, err = addEntry(newAnchor(program{1, b.Program}, trID))
				if err != nil {
					return
				}
			}

			val := inp.AssetAmount()

			var issID entryRef
			issID, _, err = addEntry(newIssuance(nonceHash, val, inpRefdataID))
			if err != nil {
				return
			}

			muxSources[i] = valueSource{
				Ref:      issID,
				Position: uint64(i),
				Value:    val,
			}
		}
	}

	muxID, _, err := addEntry(newMux(muxSources))
	if err != nil {
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
				return
			}
		}

		var resultID entryRef
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			resultID, _, err = addEntry(newRetirement(s, outRefdataID))
			if err != nil {
				return
			}
		} else {
			// non-retirement
			prog := program{out.VMVersion, out.ControlProgram}
			resultID, _, err = addEntry(newOutput(s, prog, outRefdataID))
			if err != nil {
				return
			}
		}

		results = append(results, resultID)
	}

	var h entry
	headerID, h, err = addEntry(newHeader(tx.Version, results, refdataID, tx.MinTime, tx.MaxTime))

	return headerID, h.(*header), entryMap, nil
}

func issuanceAnchorProg(nonce []byte, assetID bc.AssetID) program {
	b := vmutil.NewBuilder()
	b = b.AddData(nonce).AddOp(vm.OP_DROP).AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)
	return program{1, b.Program}
}
