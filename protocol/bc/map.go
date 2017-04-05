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
		firstSpend   *Spend
		firstSpendID Hash
		spends       []*Spend
		issuances    []*Issuance
		muxSources   = make([]*ValueSource, len(tx.Inputs))
	)

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*SpendInput); ok {
			prog := &Program{VmVersion: oldSp.VMVersion, Code: oldSp.ControlProgram}
			src := &ValueSource{
				Ref:      &oldSp.SourceID,
				Value:    &oldSp.AssetAmount,
				Position: oldSp.SourcePosition,
			}
			out := NewOutput(src, prog, &oldSp.RefDataHash, 0) // ordinal doesn't matter for prevouts, only for result outputs
			var prevoutID Hash
			prevoutID, err = addEntry(out)
			if err != nil {
				err = errors.Wrapf(err, "adding prevout entry for input %d", i)
				return
			}
			refdatahash := hashData(inp.ReferenceData)
			sp := NewSpend(&prevoutID, &refdatahash, uint64(i))
			sp.Witness.Arguments = oldSp.Arguments
			var id Hash
			id, err = addEntry(sp)
			if err != nil {
				err = errors.Wrapf(err, "adding spend entry for input %d", i)
				return
			}
			muxSources[i] = &ValueSource{
				Ref:   &id,
				Value: &oldSp.AssetAmount,
			}
			if firstSpend == nil {
				firstSpend = sp
				firstSpendID = id
			}
			spends = append(spends, sp)
		}
	}

	for i, inp := range tx.Inputs {
		if oldIss, ok := inp.TypedInput.(*IssuanceInput); ok {
			// Note: asset definitions, initial block ids, and issuance
			// programs are omitted here because they do not contribute to
			// the body hash of an issuance.

			var (
				anchorID    Hash
				setAnchored func(*Hash)
			)

			if len(oldIss.Nonce) == 0 {
				if firstSpend == nil {
					err = fmt.Errorf("nonce-less issuance in transaction with no spends")
					return
				}
				anchorID = firstSpendID
				setAnchored = firstSpend.SetAnchored
			} else {
				tr := NewTimeRange(tx.MinTime, tx.MaxTime)
				var trID Hash
				trID, err = addEntry(tr)
				if err != nil {
					err = errors.Wrapf(err, "adding timerange entry for input %d", i)
					return
				}

				assetID := oldIss.AssetID()

				builder := vmutil.NewBuilder()
				builder.AddData(oldIss.Nonce).AddOp(vm.OP_DROP)
				builder.AddOp(vm.OP_ASSET).AddData(assetID.Bytes()).AddOp(vm.OP_EQUAL)

				nonce := NewNonce(&Program{VmVersion: 1, Code: builder.Program}, &trID)
				var nonceID Hash
				nonceID, err = addEntry(nonce)
				if err != nil {
					err = errors.Wrapf(err, "adding nonce entry for input %d", i)
					return
				}
				anchorID = nonceID
				setAnchored = nonce.SetAnchored
			}

			val := inp.AssetAmount()

			refdatahash := hashData(inp.ReferenceData)
			assetdefhash := hashData(oldIss.AssetDefinition)
			iss := NewIssuance(&anchorID, &val, &refdatahash, uint64(i))
			iss.Witness.AssetDefinition = &AssetDefinition{
				InitialBlockId: &oldIss.InitialBlock,
				Data:           &assetdefhash,
				IssuanceProgram: &Program{
					VmVersion: oldIss.VMVersion,
					Code:      oldIss.IssuanceProgram,
				},
			}
			iss.Witness.Arguments = oldIss.Arguments
			var issID Hash
			issID, err = addEntry(iss)
			if err != nil {
				err = errors.Wrapf(err, "adding issuance entry for input %d", i)
				return
			}

			setAnchored(&issID)

			muxSources[i] = &ValueSource{
				Ref:   &issID,
				Value: &val,
			}
			issuances = append(issuances, iss)
		}
	}

	mux := NewMux(muxSources, &Program{VmVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	var muxID Hash
	muxID, err = addEntry(mux)
	if err != nil {
		err = errors.Wrap(err, "adding mux entry")
		return
	}

	for _, sp := range spends {
		spentOutput := entryMap[*sp.Body.SpentOutputId].(*Output)
		sp.SetDestination(&muxID, spentOutput.Body.Source.Value, sp.Ordinal)
	}
	for _, iss := range issuances {
		iss.SetDestination(&muxID, iss.Body.Value, iss.Ordinal)
	}

	var resultIDs []*Hash

	for i, out := range tx.Outputs {
		src := &ValueSource{
			Ref:      &muxID,
			Value:    &out.AssetAmount,
			Position: uint64(i),
		}
		var dest *ValueDestination
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			refdatahash := hashData(out.ReferenceData)
			r := NewRetirement(src, &refdatahash, uint64(i))
			var rID Hash
			rID, err = addEntry(r)
			if err != nil {
				err = errors.Wrapf(err, "adding retirement entry for output %d", i)
				return
			}
			resultIDs = append(resultIDs, &rID)
			dest = &ValueDestination{
				Ref:      &rID,
				Position: 0,
			}
		} else {
			// non-retirement
			prog := &Program{out.VMVersion, out.ControlProgram}
			refdatahash := hashData(out.ReferenceData)
			o := NewOutput(src, prog, &refdatahash, uint64(i))
			var oID Hash
			oID, err = addEntry(o)
			if err != nil {
				err = errors.Wrapf(err, "adding output entry for output %d", i)
				return
			}
			resultIDs = append(resultIDs, &oID)
			dest = &ValueDestination{
				Ref:      &oID,
				Position: 0,
			}
		}
		dest.Value = src.Value
		mux.Witness.Destinations = append(mux.Witness.Destinations, dest)
	}

	refdatahash := hashData(tx.ReferenceData)
	h := NewTxHeader(tx.Version, resultIDs, &refdatahash, tx.MinTime, tx.MaxTime)
	headerID, err = addEntry(h)
	if err != nil {
		err = errors.Wrap(err, "adding header entry")
		return
	}

	return headerID, h, entryMap, nil
}

func mapBlockHeader(old *BlockHeader) (bhID Hash, bh *BlockHeaderEntry) {
	bh = NewBlockHeaderEntry(old.Version, old.Height, &old.PreviousBlockHash, old.TimestampMS, &old.TransactionsMerkleRoot, &old.AssetsMerkleRoot, old.ConsensusProgram)
	bh.Witness.Arguments = old.Witness
	bhID = EntryID(bh)
	return
}

func MapBlock(old *Block) *BlockEntries {
	if old == nil {
		return nil // if old is nil, so should new be
	}
	b := new(BlockEntries)
	b.ID, b.BlockHeaderEntry = mapBlockHeader(&old.BlockHeader)
	for _, oldTx := range old.Transactions {
		b.Transactions = append(b.Transactions, oldTx.TxEntries)
	}
	return b
}

func hashData(data []byte) Hash {
	var b32 [32]byte
	sha3pool.Sum256(b32[:], data)
	return NewHash(b32)
}
