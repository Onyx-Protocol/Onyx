package tx

import (
	"chain/protocol/bc"
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

	hdr.body.data = refdataID

	addMuxSource := func(e entry, val bc.AssetAmount) error {
		id, err := entryID(e)
		if err != nil {
			return err
		}
		s := valueSource{
			ref:      id,
			position: len(muxSources),
			value:    val,
		}
		muxSources = append(muxSources, s)
		return nil
	}

	if len(tx.ReferenceData) > 0 {
		d := newData(hashData(tx.ReferenceData))
		entries = append(entries, d)
		references = append(references, d.ID())
	}

	for _, inp := range tx.Inputs {
		if inp.IsIssuance() {
			oldIss := inp.TypedInput.(*bc.IssuanceInput)

			var (
				anchorHash entryRef
				aDefHash   entryRef
			)

			if len(oldIss.Nonce) == 0 {
				// xxx anchorHash = "first spend input of the oldtx" (does this mean the txhash of the prevout of the spend?)
			} else {
				prog := xxx // Program{VM1, PUSHDATA(oldIss.Nonce)}
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
				issProg := program{oldIss.VMVersion, oldIss.IssuanceProgram}

				if len(oldIss.AssetDefinition) > 0 {
					adef := newData(hashData(oldIss.AssetDefinition))
					entries = append(entries, adef)
				}
			}

			val := inp.AssetAmount()

			iss := newIssuance(inp.AssetAmount(), oldIss.InitialBlock, issProg, oldIss.Arguments, anchorHash, aDefHash)
			entries = append(entries, iss)

			err = addMuxSource(iss, val)
			if err != nil {
				return nil, nil, err
			}
		} else {
			oldSp := inp.TypedInput.(*bc.SpendInput)
			sp := newSpend(oldSp.OutputID, oldSp.Arguments, entryRef{}) // last arg is the refdata entryref
			entries = append(entries, sp)

			err = addMuxSource(sp, oldSp.AssetAmount)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	mux := newMux(muxSources)
	muxID, err := entryID(mux)
	if err != nil {
		return nil, nil, err
	}

	for i, out := range tx.Outputs {
		s := valueSource{
			ref:      muxID,
			position: i,
			value:    out.AssetAmount,
		}

		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
		} else {
			// non-retirement
		}
	}

	header = newHeader(tx.Version, results, refdataID, references, tx.MinTime, tx.MaxTime)
}
