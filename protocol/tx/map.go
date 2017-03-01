package tx

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func mapTx(tx *bc.TxData) (headerID bc.Hash, hdr *header, entryMap map[bc.Hash]entry, err error) {
	entryMap = make(map[bc.Hash]entry)

	addEntry := func(e entry) (w *idWrapper, err error) {
		defer func() {
			if pErr, ok := recover().(error); ok {
				err = pErr
			}
		}()
		w = newIDWrapper(e, nil)
		entryMap[w.Hash] = w
		return w, err
	}

	// Loop twice over tx.Inputs, once for spends and once for
	// issuances.  Do spends first so the entry ID of the first spend is
	// available in case an issuance needs it for its anchor.

	var firstSpend *spend
	muxSources := make([]valueSource, len(tx.Inputs))

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*bc.SpendInput); ok {
			prog := program{VMVersion: oldSp.VMVersion, Code: oldSp.ControlProgram}
			out := newOutput(prog, oldSp.RefDataHash, 0) // ordinal doesn't matter for prevouts, only for result outputs
			out.setSourceID(oldSp.SourceID, oldSp.AssetAmount, oldSp.SourcePosition)
			sp := newSpend(out, hashData(inp.ReferenceData), i)
			var w *idWrapper
			w, err = addEntry(sp)
			if err != nil {
				err = errors.Wrapf(err, "adding spend entry for input %d", i)
				return
			}
			muxSources[i] = valueSource{
				Ref:   w.Hash,
				Value: oldSp.AssetAmount,
			}
			if firstSpend == nil {
				firstSpend = sp
			}
		}
	}

	for i, inp := range tx.Inputs {
		if oldIss, ok := inp.TypedInput.(*bc.IssuanceInput); ok {
			// Note: asset definitions, initial block ids, and issuance
			// programs are omitted here because they do not contribute to
			// the body hash of an issuance.

			var nonce entry

			if len(oldIss.Nonce) == 0 {
				if firstSpend == nil {
					err = fmt.Errorf("nonce-less issuance in transaction with no spends")
					return
				}
				nonce = firstSpend
			} else {
				tr := newTimeRange(tx.MinTime, tx.MaxTime)
				_, err = addEntry(tr)
				if err != nil {
					err = errors.Wrapf(err, "adding timerange entry for input %d", i)
					return
				}

				assetID := oldIss.AssetID()
				b := vmutil.NewBuilder()
				b = b.AddData(oldIss.Nonce).AddOp(vm.OP_DROP).AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)

				nonce = newNonce(program{1, b.Program}, tr)
				_, err = addEntry(nonce)
				if err != nil {
					err = errors.Wrapf(err, "adding nonce entry for input %d", i)
					return
				}
			}

			val := inp.AssetAmount()

			iss := newIssuance(nonce, val, hashData(inp.ReferenceData), i)
			var w *idWrapper
			w, err = addEntry(iss)
			if err != nil {
				err = errors.Wrapf(err, "adding issuance entry for input %d", i)
				return
			}

			muxSources[i] = valueSource{
				Ref:   w.Hash,
				Value: val,
			}
		}
	}

	mux := newMux(program{VMVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	for _, src := range muxSources {
		// TODO(bobg): addSource will recompute the hash of
		// entryMap[src.Ref], which is already available as src.Ref - fix
		// this (and a number of other such places)
		mux.addSource(entryMap[src.Ref], src.Value, src.Position)
	}
	_, err = addEntry(mux)
	if err != nil {
		err = errors.Wrap(err, "adding mux entry")
		return
	}

	var results []entry

	for i, out := range tx.Outputs {
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			r := newRetirement(hashData(out.ReferenceData), i)
			r.setSource(mux, out.AssetAmount, uint64(i))
			_, err = addEntry(r)
			if err != nil {
				err = errors.Wrapf(err, "adding retirement entry for output %d", i)
				return
			}
			results = append(results, r)
		} else {
			// non-retirement
			prog := program{out.VMVersion, out.ControlProgram}
			o := newOutput(prog, hashData(out.ReferenceData), i)
			o.setSource(mux, out.AssetAmount, uint64(i))
			_, err = addEntry(o)
			if err != nil {
				err = errors.Wrapf(err, "adding output entry for output %d", i)
				return
			}
			results = append(results, o)
		}
	}

	h := newHeader(tx.Version, hashData(tx.ReferenceData), tx.MinTime, tx.MaxTime)
	for _, res := range results {
		h.addResult(res)
	}
	var w *idWrapper
	w, err = addEntry(h)
	if err != nil {
		err = errors.Wrap(err, "adding header entry")
		return
	}

	return w.Hash, h, entryMap, nil
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
