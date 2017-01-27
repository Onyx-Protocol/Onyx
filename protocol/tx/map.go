package tx

import (
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func mapTx(tx *bc.TxData) (headerID entryRef, hdr *header, entryMap map[entryRef]entry, err error) {
	var (
		muxSources []valueSource
		refdataID  entryRef
	)

	entryMap = make(map[entryRef]entry)

	addEntry := func(e entry) (id entryRef, entry entry, err error) {
		id, err = entryID(e)
		if err != nil {
			return
		}
		entryMap[id] = e
		return id, e, nil
	}

	addMuxSource := func(id entryRef, val bc.AssetAmount) {
		s := valueSource{
			Ref:      id,
			Position: uint64(len(muxSources)),
			Value:    val,
		}
		muxSources = append(muxSources, s)
	}

	if len(tx.ReferenceData) > 0 {
		refdataID, _, err = addEntry(newData(hashData(tx.ReferenceData)))
		if err != nil {
			return
		}
	}

	for _, inp := range tx.Inputs {
		var inpRefdataID entryRef
		if len(inp.ReferenceData) > 0 {
			inpRefdataID, _, err = addEntry(newData(hashData(inp.ReferenceData)))
			if err != nil {
				return
			}
		}

		if inp.IsIssuance() {
			// xxx asset definitions, initial block ids, and issuance
			// programs are omitted here because they do not contribute to
			// the body hash of an issuance.

			oldIss := inp.TypedInput.(*bc.IssuanceInput)

			var nonceHash entryRef

			if len(oldIss.Nonce) == 0 {
				// xxx nonceHash = "first spend input of the oldtx" (does this mean the txhash of the prevout of the spend?)
				// xxx Oleg: We need to locate one of the new inputs and take the first one's ID here. 
				//           But if none are mapped yet, we need to remember this issuance and get back to it when such input is mapped.
			} else {
				prog := issuanceAnchorProg(oldIss.Nonce, oldIss.AssetID())

				var trID entryRef
				trID, _, err = addEntry(newTimeRange(tx.MinTime, tx.MaxTime))
				if err != nil {
					return
				}

				nonceHash, _, err = addEntry(newNonce(prog, trID))
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

			addMuxSource(issID, val)
		} else {
			oldSp := inp.TypedInput.(*bc.SpendInput)

			var spID entryRef
			spID, _, err = addEntry(newSpend(entryRef(oldSp.SpentOutputID.Hash), inpRefdataID))
			if err != nil {
				return
			}

			addMuxSource(spID, oldSp.AssetAmount)
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
