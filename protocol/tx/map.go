package tx

import (
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func mapTx(tx *bc.TxData) (hdr *header, entryMap map[entryRef]entry, err error) {
	var (
		references []entryRef
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
			return nil, nil, err
		}
		references = append(references, refdataID)
	}

	for _, inp := range tx.Inputs {
		var inpRefdataID entryRef
		if len(inp.ReferenceData) > 0 {
			inpRefdataID, _, err = addEntry(newData(hashData(inp.ReferenceData)))
			if err != nil {
				return nil, nil, err
			}
		}

		if inp.IsIssuance() {
			oldIss := inp.TypedInput.(*bc.IssuanceInput)

			var anchorHash entryRef

			if len(oldIss.Nonce) == 0 {
				// xxx anchorHash = "first spend input of the oldtx" (does this mean the txhash of the prevout of the spend?)
			} else {
				prog := issuanceAnchorProg(oldIss.Nonce, oldIss.AssetID(), oldIss.VMVersion)

				trID, _, err := addEntry(newTimeRange(tx.MinTime, tx.MaxTime))
				if err != nil {
					return nil, nil, err
				}

				anchorHash, _, err = addEntry(newAnchor(prog, trID))
				if err != nil {
					return nil, nil, err
				}

				// xxx asset definitions omitted from entryMap; not needed for body hashing
			}

			val := inp.AssetAmount()

			issID, _, err := addEntry(newIssuance(anchorHash, val, inpRefdataID))
			if err != nil {
				return nil, nil, err
			}

			addMuxSource(issID, val)
		} else {
			oldSp := inp.TypedInput.(*bc.SpendInput)

			spID, _, err := addEntry(newSpend(entryRef(oldSp.SpentOutputID.Hash), inpRefdataID))
			if err != nil {
				return nil, nil, err
			}

			addMuxSource(spID, oldSp.AssetAmount)
		}
	}

	muxID, _, err := addEntry(newMux(muxSources))
	if err != nil {
		return nil, nil, err
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
				return nil, nil, err
			}
		}

		var resultID entryRef
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			resultID, _, err = addEntry(newRetirement(s, outRefdataID))
			if err != nil {
				return nil, nil, err
			}
		} else {
			// non-retirement
			prog := program{out.VMVersion, out.ControlProgram}
			resultID, _, err = addEntry(newOutput(s, prog, outRefdataID))
			if err != nil {
				return nil, nil, err
			}
		}

		results = append(results, resultID)
	}

	var h entry
	_, h, err = addEntry(newHeader(tx.Version, results, refdataID, references, tx.MinTime, tx.MaxTime))

	return h.(*header), entryMap, nil
}

func issuanceAnchorProg(nonce []byte, assetID bc.AssetID, vmVersion uint64) program {
	b := vmutil.NewBuilder()
	b = b.AddData(nonce).AddOp(vm.OP_DROP).AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)
	return program{vmVersion, b.Program}
}
