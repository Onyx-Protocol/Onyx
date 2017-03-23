package bc

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func mapTx(tx *TxData) (headerID Hash, hdr *TxHeader, entryMap map[Hash]Entry, err error) {
	entryMap = make(map[Hash]Entry)

	addEntry := func(e Entry) (id Hash, err error) {
		defer func() {
			if pErr, ok := recover().(error); ok {
				err = pErr
			}
		}()
		id = EntryID(e)
		entryMap[id] = e
		return id, err
	}

	// Loop twice over tx.Inputs, once for spends and once for
	// issuances.  Do spends first so the entry ID of the first spend is
	// available in case an issuance needs it for its anchor.

	var (
		firstSpend *Spend
		spends     []*Spend
		issuances  []*Issuance
		muxSources = make([]ValueSource, len(tx.Inputs))
	)

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*SpendInput); ok {
			prog := Program{VMVersion: oldSp.VMVersion, Code: oldSp.ControlProgram}
			src := ValueSource{
				Ref:      oldSp.SourceID,
				Value:    oldSp.AssetAmount,
				Position: oldSp.SourcePosition,
			}
			out := NewOutput(src, prog, oldSp.RefDataHash, 0) // ordinal doesn't matter for prevouts, only for result outputs
			sp := NewSpend(out, hashData(inp.ReferenceData), i)
			var id Hash
			id, err = addEntry(sp)
			if err != nil {
				err = errors.Wrapf(err, "adding spend entry for input %d", i)
				return
			}
			muxSources[i] = ValueSource{
				Ref:   id,
				Value: oldSp.AssetAmount,
				Entry: sp,
			}
			if firstSpend == nil {
				firstSpend = sp
			}
			spends = append(spends, sp)
		}
	}

	for i, inp := range tx.Inputs {
		if oldIss, ok := inp.TypedInput.(*IssuanceInput); ok {
			// Note: asset definitions, initial block ids, and issuance
			// programs are omitted here because they do not contribute to
			// the body hash of an issuance.

			var nonce Entry

			if len(oldIss.Nonce) == 0 {
				if firstSpend == nil {
					err = fmt.Errorf("nonce-less issuance in transaction with no spends")
					return
				}
				nonce = firstSpend
			} else {
				tr := NewTimeRange(tx.MinTime, tx.MaxTime)
				_, err = addEntry(tr)
				if err != nil {
					err = errors.Wrapf(err, "adding timerange entry for input %d", i)
					return
				}

				assetID := oldIss.AssetID()

				builder := vmutil.NewBuilder()
				builder.AddData(oldIss.Nonce).AddOp(vm.OP_DROP)
				builder.AddOp(vm.OP_ASSET).AddData(assetID[:]).AddOp(vm.OP_EQUAL)

				nonce = NewNonce(Program{VMVersion: 1, Code: builder.Program}, tr)
				_, err = addEntry(nonce)
				if err != nil {
					err = errors.Wrapf(err, "adding nonce entry for input %d", i)
					return
				}
			}

			val := inp.AssetAmount()

			iss := NewIssuance(nonce, val, hashData(inp.ReferenceData), i)
			var issID Hash
			issID, err = addEntry(iss)
			if err != nil {
				err = errors.Wrapf(err, "adding issuance entry for input %d", i)
				return
			}

			muxSources[i] = ValueSource{
				Ref:   issID,
				Value: val,
				Entry: iss,
			}
			issuances = append(issuances, iss)
		}
	}

	mux := NewMux(muxSources, Program{VMVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	var muxID Hash
	muxID, err = addEntry(mux)
	if err != nil {
		err = errors.Wrap(err, "adding mux entry")
		return
	}

	var results []Entry

	for i, out := range tx.Outputs {
		src := ValueSource{
			Ref:      muxID,
			Value:    out.AssetAmount,
			Position: uint64(i),
			Entry:    mux,
		}
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			r := NewRetirement(src, hashData(out.ReferenceData), i)
			_, err = addEntry(r)
			if err != nil {
				err = errors.Wrapf(err, "adding retirement entry for output %d", i)
				return
			}
			results = append(results, r)
		} else {
			// non-retirement
			prog := Program{out.VMVersion, out.ControlProgram}
			o := NewOutput(src, prog, hashData(out.ReferenceData), i)
			_, err = addEntry(o)
			if err != nil {
				err = errors.Wrapf(err, "adding output entry for output %d", i)
				return
			}
			results = append(results, o)
		}
	}

	h := NewTxHeader(tx.Version, results, hashData(tx.ReferenceData), tx.MinTime, tx.MaxTime)
	headerID, err = addEntry(h)
	if err != nil {
		err = errors.Wrap(err, "adding header entry")
		return
	}

	return headerID, h, entryMap, nil
}

func mapBlockHeader(old *BlockHeader) (bhID Hash, bh *BlockHeaderEntry) {
	bh = NewBlockHeaderEntry(old.Version, old.Height, old.PreviousBlockHash, old.TimestampMS, old.TransactionsMerkleRoot, old.AssetsMerkleRoot, old.ConsensusProgram)
	bhID = EntryID(bh)
	return
}

func hashData(data []byte) (h Hash) {
	sha3pool.Sum256(h[:], data)
	return
}
