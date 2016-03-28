package fedchain

import (
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"

	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
	"chain/fedchain/validation"
	"chain/log"
	"chain/net/trace/span"
)

// maxBlockTxs limits the number of transactions
// included in each block.
const maxBlockTxs = 10000

// ErrBadBlock is returned when a block is invalid.
var ErrBadBlock = errors.New("invalid block")

// GenerateBlock generates a valid, but unsigned, candidate block from
// the current tx pool.  It returns the new block and the previous
// block (the latest on the blockchain).  It has no side effects.
func (fc *FC) GenerateBlock(ctx context.Context, now time.Time) (b, prev *bc.Block, err error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	ts := uint64(now.Unix())

	prev, err = fc.store.LatestBlock(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fetch latest block")
	}

	if ts < prev.Timestamp {
		return nil, nil, errors.New("timestamp is earlier than prevblock timestamp")
	}

	txs, err := fc.store.PoolTxs(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get pool TXs")
	}
	if len(txs) > maxBlockTxs {
		txs = txs[:maxBlockTxs]
	}

	b = &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           bc.NewBlockVersion,
			Height:            prev.Height + 1,
			PreviousBlockHash: prev.Hash(),

			// TODO: Calculate merkle hash of blockchain state.
			//StateRoot:

			Timestamp: ts,

			// TODO: Generate SignatureScript
			OutputScript: prev.OutputScript,
		},
	}

	poolView := state.NewMemView()
	bcView, err := fc.store.NewViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}
	view := state.Compose(poolView, bcView)
	ctx = span.NewContextSuffix(ctx, "-validate-all")
	defer span.Finish(ctx)
	for _, tx := range txs {
		if validation.ValidateTxInputs(ctx, view, tx) == nil {
			validation.ApplyTx(ctx, view, tx)
			b.Transactions = append(b.Transactions, tx)
		}
	}

	b.TxRoot = validation.CalcMerkleRoot(b.Transactions)

	return b, prev, nil
}

// AddBlock validates block and (if valid) adds it to the chain.
// It also deletes any pending transactions that become conflicted
// as a result of this block.
//
// This updates the UTXO set and ADPs, and calls new-block callbacks.
func (fc *FC) AddBlock(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	bcView, err := fc.store.NewViewForPrevouts(ctx, block.Transactions)
	if err != nil {
		return errors.Wrap(err, "txdb")
	}
	mv := state.NewMemView()

	view := state.Compose(mv, bcView)
	err = fc.validateBlock(ctx, block, view)
	if err != nil {
		return errors.Wrap(err, "block validation")
	}

	newTxs, conflicts, err := fc.applyBlock(ctx, block, mv)
	if err != nil {
		return errors.Wrap(err, "applying block")
	}

	for _, tx := range newTxs {
		for _, cb := range fc.txCallbacks {
			cb(ctx, tx)
		}
	}

	for _, cb := range fc.blockCallbacks {
		cb(ctx, block, conflicts)
	}

	fc.height.cond.L.Lock()
	defer fc.height.cond.L.Unlock()

	fc.height.n = block.Height
	fc.height.cond.Broadcast()

	// TODO(kr): add WaitTx method and notify any waiting goroutines here.
	// See https://github.com/chain-engineering/chain/pull/480 for a sketch.
	return nil
}

// ValidateBlockForSig performs validation on an incoming _unsigned_
// block in preparation for signing it.  By definition it does not
// execute the sigscript.
func (fc *FC) ValidateBlockForSig(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	bcView, err := fc.store.NewViewForPrevouts(ctx, block.Transactions)
	if err != nil {
		return errors.Wrap(err, "txdb")
	}
	mv := state.NewMemView()

	prevBlock, err := fc.LatestBlock(ctx)
	if err != nil && errors.Root(err) != ErrNoBlocks {
		return errors.Wrap(err, "getting latest known block")
	}

	err = validation.ValidateBlockForSig(ctx, state.Compose(mv, bcView), prevBlock, block)
	return errors.Wrap(err, "validation")
}

// validateBlock performs validation on an incoming block, in advance of
// applying the block to the store.
func (fc *FC) validateBlock(ctx context.Context, block *bc.Block, view state.View) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	prevBlock, err := fc.store.LatestBlock(ctx)
	if err != nil {
		return errors.Wrap(err, "loading previous block")
	}
	err = validation.ValidateBlockHeader(ctx, prevBlock, block)
	if err != nil {
		return errors.Wrap(err, "validating block header")
	}

	if isSignedByTrustedHost(block, fc.trustedKeys) {
		err = validation.ApplyBlock(ctx, view, block)
	} else {
		err = validation.ValidateAndApplyBlock(ctx, view, prevBlock, block)
	}
	if err != nil {
		return errors.Wrapf(ErrBadBlock, "validate block: %v", err)
	}
	return nil
}

func isSignedByTrustedHost(block *bc.Block, trustedKeys []*btcec.PublicKey) bool {
	sigs, err := txscript.PushedData(block.SignatureScript)
	if err != nil {
		return false
	}

	hash := block.HashForSig()
	for _, sig := range sigs {
		if len(sig) == 0 {
			continue
		}
		parsedSig, err := btcec.ParseSignature(sig, btcec.S256())
		if err != nil { // could be arbitrary push data
			continue
		}
		for _, pubk := range trustedKeys {
			if parsedSig.Verify(hash[:], pubk) {
				return true
			}
		}
	}

	return false
}

func (fc *FC) applyBlock(ctx context.Context, block *bc.Block, mv *state.MemView) (newTxs []*bc.Tx, conflictingTxs []*bc.Tx, err error) {
	delta := make([]*state.Output, 0, len(mv.Outs))
	for _, out := range mv.Outs {
		delta = append(delta, out)
	}

	newTxs, err = fc.store.ApplyBlock(ctx, block, mv.ADPs, delta, mv.Issuance, mv.Destroyed)
	if err != nil {
		return nil, nil, errors.Wrap(err, "storing block")
	}

	conflicts, err := fc.rebuildPool(ctx, block)
	return newTxs, conflicts, errors.Wrap(err, "rebuilding pool")
}

func (fc *FC) rebuildPool(ctx context.Context, block *bc.Block) ([]*bc.Tx, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	txInBlock := make(map[bc.Hash]bool)
	for _, tx := range block.Transactions {
		txInBlock[tx.Hash] = true
	}

	var (
		conflictTxs  []*bc.Tx
		confirmedTxs []*bc.Tx
	)

	txs, err := fc.store.PoolTxs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	poolView := state.NewMemView()
	bcView, err := fc.store.NewViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, errors.Wrap(err, "blockchain view")
	}
	view := state.Compose(poolView, bcView)
	for _, tx := range txs {
		txErr := validation.ValidateTxInputs(ctx, view, tx)

		for _, out := range tx.Outputs {
			if _, ok := poolView.Issuance[out.AssetID]; !ok {
				poolView.Issuance[out.AssetID] = 0
			}
			if _, ok := poolView.Destroyed[out.AssetID]; !ok {
				poolView.Destroyed[out.AssetID] = 0
			}
		}

		// Have to explicitly check that tx is not in block
		// because issuance transactions are always valid, even duplicates.
		// TODO(erykwalder): Remove this check when issuances become unique
		if txErr == nil && !txInBlock[tx.Hash] {
			validation.ApplyTx(ctx, view, tx)
		} else {
			if txInBlock[tx.Hash] {
				confirmedTxs = append(confirmedTxs, tx)
				continue
			}
			conflictTxs = append(conflictTxs, tx)
			// This should never happen in sandbox, unless a reservation expired
			// before the original tx was finalized.
			log.Messagef(ctx, "deleting conflict tx %v because %q", tx.Hash, txErr)
			for i, in := range tx.Inputs {
				out := view.Output(ctx, in.Previous)
				if out == nil {
					log.Messagef(ctx, "conflict tx %v missing input %d (%v)", tx.Hash, in.Previous)
					continue
				}
				if out.Spent {
					log.Messagef(ctx, "conflict tx %v spent input %d (%v) inblock=%v inpool=%v",
						tx.Hash, i, in.Previous, bcView.Output(ctx, in.Previous), poolView.Output(ctx, in.Previous))
				}
			}
		}
	}

	err = fc.store.CleanPool(ctx, confirmedTxs, conflictTxs, poolView.Issuance, poolView.Destroyed)
	if err != nil {
		return nil, errors.Wrap(err, "removing conflicting txs")
	}

	return conflictTxs, nil
}

// ComputeBlockSignature signs a block with the given key.  It does
// not validate the block.
func ComputeBlockSignature(b *bc.Block, key *btcec.PrivateKey) (*btcec.Signature, error) {
	hash := b.HashForSig()
	return key.Sign(hash[:])
}

// AddSignaturesToBlock adds signatures to a block, replacing the
// block's SignatureScript.  The signatures must be in the correct
// order, to wit: matching the order of pubkeys in the previous
// block's output script.
func AddSignaturesToBlock(b *bc.Block, signatures []*btcec.Signature) error {
	// assumes multisig output script
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_0) // required because of bug in OP_CHECKMULTISIG
	for _, signature := range signatures {
		serialized := signature.Serialize()
		serialized = append(serialized, 1) // append hashtype -- unused for blocks
		builder.AddData(serialized)
	}
	script, err := builder.Script()
	if err != nil {
		return errors.Wrap(err, "finalizing block sigscript")
	}

	b.SignatureScript = script

	return nil
}

// GenerateBlockScript generates a predicate script
// requiring nSigs signatures from the given keys.
func GenerateBlockScript(keys []*btcec.PublicKey, nSigs int) ([]byte, error) {
	var addrs []*btcutil.AddressPubKey
	for _, key := range keys {
		keyData := key.SerializeCompressed()
		addr, err := btcutil.NewAddressPubKey(keyData, &chaincfg.MainNetParams)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return txscript.MultiSigScript(addrs, nSigs)
}

// UpsertGenesisBlock creates a genesis block iff it does not exist.
func (fc *FC) UpsertGenesisBlock(ctx context.Context, pubkeys []*btcec.PublicKey, nSigs int) (*bc.Block, error) {
	// TODO(bobg): Cache the genesis block if it exists and return it
	// rather than always consing up a new one.
	script, err := GenerateBlockScript(pubkeys, nSigs)
	if err != nil {
		return nil, err
	}
	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:      bc.NewBlockVersion,
			Height:       1,
			Timestamp:    uint64(time.Now().Unix()),
			OutputScript: script,
		},
	}

	latestBlock, err := fc.store.LatestBlock(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting latest block")
	}
	if latestBlock == nil {
		err = fc.AddBlock(ctx, b)
		if err != nil {
			return nil, errors.Wrap(err, "adding genesis block")
		}
	}

	return b, nil
}
