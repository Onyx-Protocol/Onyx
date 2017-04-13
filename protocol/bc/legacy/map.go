package legacy

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

// MapTx converts a legacy TxData object into its entries-based
// representation.
func MapTx(oldTx *TxData) (txEntries *bc.Tx, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()

	txid, header, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, errors.Wrap(err, "mapping old transaction to new")
	}

	txEntries = &bc.Tx{
		TxHeader: header,
		ID:       txid,
		Entries:  entries,
		TxInputs: make([]bc.Entry, len(oldTx.Inputs)),
		InputIDs: make([]bc.Hash, len(oldTx.Inputs)),
	}

	var (
		nonceIDs       = make(map[bc.Hash]bool)
		spentOutputIDs = make(map[bc.Hash]bool)
		outputIDs      = make(map[bc.Hash]bool)
	)

	for id, e := range entries {
		var ord uint64
		switch e := e.(type) {
		case *bc.Issuance:
			anchor, ok := entries[*e.Body.AnchorId]
			if !ok {
				return nil, fmt.Errorf("entry for anchor ID %x not found", e.Body.AnchorId.Bytes())
			}
			if _, ok := anchor.(*bc.Nonce); ok {
				nonceIDs[*e.Body.AnchorId] = true
			}
			ord = e.Ordinal
			// resume below after the switch

		case *bc.Spend:
			spentOutputIDs[*e.Body.SpentOutputId] = true
			ord = e.Ordinal
			// resume below after the switch

		case *bc.Output:
			outputIDs[id] = true
			continue

		default:
			continue
		}
		if ord >= uint64(len(oldTx.Inputs)) {
			return nil, fmt.Errorf("%T entry has out-of-range ordinal %d", e, ord)
		}
		txEntries.TxInputs[ord] = e
		txEntries.InputIDs[ord] = id
	}

	for id := range nonceIDs {
		txEntries.NonceIDs = append(txEntries.NonceIDs, id)
	}
	for id := range spentOutputIDs {
		txEntries.SpentOutputIDs = append(txEntries.SpentOutputIDs, id)
	}
	for id := range outputIDs {
		txEntries.OutputIDs = append(txEntries.OutputIDs, id)
	}

	return txEntries, nil
}

func mapTx(tx *TxData) (headerID bc.Hash, hdr *bc.TxHeader, entryMap map[bc.Hash]bc.Entry, err error) {
	entryMap = make(map[bc.Hash]bc.Entry)

	addEntry := func(e bc.Entry) (id bc.Hash, err error) {
		defer func() {
			if pErr, ok := recover().(error); ok {
				err = pErr
			}
		}()
		id = bc.EntryID(e)
		entryMap[id] = e
		return id, err
	}

	// Loop twice over tx.Inputs, once for spends and once for
	// issuances.  Do spends first so the entry ID of the first spend is
	// available in case an issuance needs it for its anchor.

	var (
		firstSpend   *bc.Spend
		firstSpendID bc.Hash
		spends       []*bc.Spend
		issuances    []*bc.Issuance
		muxSources   = make([]*bc.ValueSource, len(tx.Inputs))
	)

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*SpendInput); ok {
			prog := &bc.Program{VmVersion: oldSp.VMVersion, Code: oldSp.ControlProgram}
			src := &bc.ValueSource{
				Ref:      &oldSp.SourceID,
				Value:    &oldSp.AssetAmount,
				Position: oldSp.SourcePosition,
			}
			out := bc.NewOutput(src, prog, &oldSp.RefDataHash, 0) // ordinal doesn't matter for prevouts, only for result outputs
			var prevoutID bc.Hash
			prevoutID, err = addEntry(out)
			if err != nil {
				err = errors.Wrapf(err, "adding prevout entry for input %d", i)
				return
			}
			refdatahash := hashData(inp.ReferenceData)
			sp := bc.NewSpend(&prevoutID, &refdatahash, uint64(i))
			sp.Witness.Arguments = oldSp.Arguments
			var id bc.Hash
			id, err = addEntry(sp)
			if err != nil {
				err = errors.Wrapf(err, "adding spend entry for input %d", i)
				return
			}
			muxSources[i] = &bc.ValueSource{
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
				anchorID    bc.Hash
				setAnchored func(*bc.Hash)
			)

			if len(oldIss.Nonce) == 0 {
				if firstSpend == nil {
					err = fmt.Errorf("nonce-less issuance in transaction with no spends")
					return
				}
				anchorID = firstSpendID
				setAnchored = firstSpend.SetAnchored
			} else {
				tr := bc.NewTimeRange(tx.MinTime, tx.MaxTime)
				var trID bc.Hash
				trID, err = addEntry(tr)
				if err != nil {
					err = errors.Wrapf(err, "adding timerange entry for input %d", i)
					return
				}

				assetID := oldIss.AssetID()

				builder := vmutil.NewBuilder()
				builder.AddData(oldIss.Nonce).AddOp(vm.OP_DROP)
				builder.AddOp(vm.OP_ASSET).AddData(assetID.Bytes()).AddOp(vm.OP_EQUAL)

				nonce := bc.NewNonce(&bc.Program{VmVersion: 1, Code: builder.Program}, &trID)
				var nonceID bc.Hash
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
			iss := bc.NewIssuance(&anchorID, &val, &refdatahash, uint64(i))
			iss.Witness.AssetDefinition = &bc.AssetDefinition{
				InitialBlockId: &oldIss.InitialBlock,
				Data:           &assetdefhash,
				IssuanceProgram: &bc.Program{
					VmVersion: oldIss.VMVersion,
					Code:      oldIss.IssuanceProgram,
				},
			}
			iss.Witness.Arguments = oldIss.Arguments
			var issID bc.Hash
			issID, err = addEntry(iss)
			if err != nil {
				err = errors.Wrapf(err, "adding issuance entry for input %d", i)
				return
			}

			setAnchored(&issID)

			muxSources[i] = &bc.ValueSource{
				Ref:   &issID,
				Value: &val,
			}
			issuances = append(issuances, iss)
		}
	}

	mux := bc.NewMux(muxSources, &bc.Program{VmVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	var muxID bc.Hash
	muxID, err = addEntry(mux)
	if err != nil {
		err = errors.Wrap(err, "adding mux entry")
		return
	}

	for _, sp := range spends {
		spentOutput := entryMap[*sp.Body.SpentOutputId].(*bc.Output)
		sp.SetDestination(&muxID, spentOutput.Body.Source.Value, sp.Ordinal)
	}
	for _, iss := range issuances {
		iss.SetDestination(&muxID, iss.Body.Value, iss.Ordinal)
	}

	var resultIDs []*bc.Hash

	for i, out := range tx.Outputs {
		src := &bc.ValueSource{
			Ref:      &muxID,
			Value:    &out.AssetAmount,
			Position: uint64(i),
		}
		var dest *bc.ValueDestination
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			refdatahash := hashData(out.ReferenceData)
			r := bc.NewRetirement(src, &refdatahash, uint64(i))
			var rID bc.Hash
			rID, err = addEntry(r)
			if err != nil {
				err = errors.Wrapf(err, "adding retirement entry for output %d", i)
				return
			}
			resultIDs = append(resultIDs, &rID)
			dest = &bc.ValueDestination{
				Ref:      &rID,
				Position: 0,
			}
		} else {
			// non-retirement
			prog := &bc.Program{out.VMVersion, out.ControlProgram}
			refdatahash := hashData(out.ReferenceData)
			o := bc.NewOutput(src, prog, &refdatahash, uint64(i))
			var oID bc.Hash
			oID, err = addEntry(o)
			if err != nil {
				err = errors.Wrapf(err, "adding output entry for output %d", i)
				return
			}
			resultIDs = append(resultIDs, &oID)
			dest = &bc.ValueDestination{
				Ref:      &oID,
				Position: 0,
			}
		}
		dest.Value = src.Value
		mux.Witness.Destinations = append(mux.Witness.Destinations, dest)
	}

	refdatahash := hashData(tx.ReferenceData)
	h := bc.NewTxHeader(tx.Version, resultIDs, &refdatahash, tx.MinTime, tx.MaxTime)
	headerID, err = addEntry(h)
	if err != nil {
		err = errors.Wrap(err, "adding header entry")
		return
	}

	return headerID, h, entryMap, nil
}

func mapBlockHeader(old *BlockHeader) (bhID bc.Hash, bh *bc.BlockHeader) {
	bh = bc.NewBlockHeader(old.Version, old.Height, &old.PreviousBlockHash, old.TimestampMS, &old.TransactionsMerkleRoot, &old.AssetsMerkleRoot, old.ConsensusProgram)
	bh.Witness.Arguments = old.Witness
	bhID = bc.EntryID(bh)
	return
}

func MapBlock(old *Block) *bc.Block {
	if old == nil {
		return nil // if old is nil, so should new be
	}
	b := new(bc.Block)
	b.ID, b.BlockHeader = mapBlockHeader(&old.BlockHeader)
	for _, oldTx := range old.Transactions {
		b.Transactions = append(b.Transactions, oldTx.Tx)
	}
	return b
}

func hashData(data []byte) bc.Hash {
	var b32 [32]byte
	sha3pool.Sum256(b32[:], data)
	return bc.NewHash(b32)
}
