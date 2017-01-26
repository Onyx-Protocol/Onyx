package tx

import (
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func mapTx(tx *bc.TxData) (header *header, entryMap map[entryRef]entry, err error) {
	var (
		entries    []entry
		references []entryRef
		muxSources []valueSource
		refdataID  entryRef
	)

	entryMap = make(map[entryRef]entry)

	if len(tx.ReferenceData) > 0 {
		refdata := newData(hashData(tx.ReferenceData))
		refdataID, err := entryID(refdata)
		if err != nil {
			return nil, nil, err
		}
		entryMap[refdataID] = refdata
	}

	addMuxSource := func(e entry, val bc.AssetAmount) error {
		id, err := entryID(e)
		if err != nil {
			return err
		}
		s := valueSource{
			Ref:      id,
			Position: len(muxSources),
			Value:    val,
		}
		muxSources = append(muxSources, s)
		return nil
	}

	if len(tx.ReferenceData) > 0 {
		d := newData(hashData(tx.ReferenceData))
		entries = append(entries, d)
		dID, err := entryID(d)
		if err != nil {
			return nil, nil, err
		}
		references = append(references, dID)
	}

	for _, inp := range tx.Inputs {
		var dataRef entryRef
		if len(inp.ReferenceData) > 0 {
			d := newData(hashData(inp.ReferenceData))
			entries = append(entries, d)
			dataRef, err = entryID(d) // xxx duplicate entry ids possible (maybe that's ok, deduping happens at the end)
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
				prog := issuanceAnchorProg(oldIss.Nonce, oldIss.AssetID())
				tr := newTimeRange(tx.MinTime, tx.MaxTime)
				entries = append(entries, tr)
				trID, err := entryID(tr)
				if err != nil {
					return nil, nil, err
				}
				a := newAnchor(prog, trID)
				anchorHash, err = entryID(a)
				if err != nil {
					return nil, nil, err
				}
				entries = append(entries, a)

				if len(oldIss.AssetDefinition) > 0 {
					adef := newData(hashData(oldIss.AssetDefinition))
					entries = append(entries, adef)
				}
			}

			val := inp.AssetAmount()

			iss := newIssuance(anchorHash, val, dataRef)
			entries = append(entries, iss)

			err = addMuxSource(iss, val)
			if err != nil {
				return nil, nil, err
			}
		} else {
			oldSp := inp.TypedInput.(*bc.SpendInput)
			sp := newSpend(entryRef(oldSp.SpentOutputID.Hash), dataRef)
			entries = append(entries, sp)

			err = addMuxSource(sp, oldSp.AssetAmount)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	mux := newMux(muxSources)
	entries = append(entries, mux)

	muxID, err := entryID(mux)
	if err != nil {
		return nil, nil, err
	}

	var results []entryRef

	for i, out := range tx.Outputs {
		s := valueSource{
			Ref:      muxID,
			Position: i,
			Value:    out.AssetAmount,
		}

		var dataID entryRef
		if len(out.ReferenceData) > 0 {
			d := newData(hashData(out.ReferenceData))
			entries = append(entries, d)
			dataID, err = entryID(d)
			if err != nil {
				return nil, nil, err
			}
		}

		var resultID entryRef
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			r := newRetirement(s, dataID)
			entries = append(entries, r)
			resultID, err = entryID(r)
			if err != nil {
				return nil, nil, err
			}
		} else {
			// non-retirement
			prog := program{1, out.ControlProgram}
			o := newOutput(s, prog, dataID)
			entries = append(entries, o)
			resultID, err = entryID(o)
			if err != nil {
				return nil, nil, err
			}
		}

		results = append(results, resultID)
	}

	header = newHeader(tx.Version, results, refdataID, references, tx.MinTime, tx.MaxTime)

	entries = append(entries, header)

	for _, e := range entries {
		id, err := entryID(e)
		if err != nil {
			return nil, nil, err
		}
		entryMap[id] = e
	}

	return header, entryMap, nil
}

func issuanceAnchorProg(nonce []byte, assetID bc.AssetID) program {
	b := vmutil.NewBuilder()
	b = b.AddData(nonce).AddOp(vm.OP_DROP).AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)
	return program{1, b.Program}
}
