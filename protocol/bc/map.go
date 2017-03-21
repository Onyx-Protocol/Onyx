package bc

import (
	"encoding/binary"
	"fmt"

	"chain/crypto/sha3pool"
	"chain/errors"
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

	var firstSpend *Spend
	muxSources := make([]valueSource, len(tx.Inputs))

	for i, inp := range tx.Inputs {
		if oldSp, ok := inp.TypedInput.(*SpendInput); ok {
			prog := Program{VMVersion: oldSp.VMVersion, Code: oldSp.ControlProgram}
			out := NewOutput(prog, oldSp.RefDataHash, 0) // ordinal doesn't matter for prevouts, only for result outputs
			out.setSourceID(oldSp.SourceID, oldSp.AssetAmount, oldSp.SourcePosition)
			sp := NewSpend(out, hashData(inp.ReferenceData), i)
			var id Hash
			id, err = addEntry(sp)
			if err != nil {
				err = errors.Wrapf(err, "adding spend entry for input %d", i)
				return
			}
			muxSources[i] = valueSource{
				Ref:   id,
				Value: oldSp.AssetAmount,
			}
			if firstSpend == nil {
				firstSpend = sp
			}
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

				// This is the program
				//   [PUSHDATA(oldIss.Nonce) DROP ASSET PUSHDATA(assetID) EQUAL]
				// minus a circular dependency on protocol/vm.
				// This code, partly duplicated from vm.PushdataBytes, will go
				// away when we're no longer mapping old txs to txentries.
				nonceLen := len(oldIss.Nonce)
				var code []byte
				switch {
				case nonceLen == 0:
					code = []byte{0}
				case nonceLen <= 75:
					code = []byte{byte(nonceLen)}
				case nonceLen < 1<<8:
					code = append([]byte{0x4c}, byte(nonceLen)) // PUSHDATA1
				case nonceLen < 1<<16:
					var b [2]byte
					binary.LittleEndian.PutUint16(b[:], uint16(nonceLen))
					code = append([]byte{0x4d}, b[:]...)
				default:
					var b [4]byte
					binary.LittleEndian.PutUint32(b[:], uint32(nonceLen))
					code = append([]byte{0x4e}, b[:]...)
				}
				code = append(code, oldIss.Nonce...)
				code = append(code, []byte{0x75, 0xc2, 0x20}...)
				code = append(code, assetID[:]...)
				code = append(code, 0x87)

				nonce = NewNonce(Program{VMVersion: 1, Code: code}, tr)
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

			muxSources[i] = valueSource{
				Ref:   issID,
				Value: val,
			}
		}
	}

	mux := NewMux(Program{VMVersion: 1, Code: []byte{0x51}}) // 0x51 == vm.OP_TRUE, minus a circular dependency on protocol/vm
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

	var results []Entry

	for i, out := range tx.Outputs {
		if isUnspendable(out.ControlProgram) {
			// retirement
			r := NewRetirement(hashData(out.ReferenceData), i)
			r.setSource(mux, out.AssetAmount, uint64(i))
			_, err = addEntry(r)
			if err != nil {
				err = errors.Wrapf(err, "adding retirement entry for output %d", i)
				return
			}
			results = append(results, r)
		} else {
			// non-retirement
			prog := Program{out.VMVersion, out.ControlProgram}
			o := NewOutput(prog, hashData(out.ReferenceData), i)
			o.setSource(mux, out.AssetAmount, uint64(i))
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

// Duplicated from vmutil.IsUnspendable to remove a circular
// dependency. Will no longer be needed when we stop mapping old txs
// to txentries.
func isUnspendable(prog []byte) bool {
	return len(prog) > 0 && prog[0] == 0x6a
}
