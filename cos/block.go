package cos

import (
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/cos/validation"
	"chain/errors"
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

	txs, err := fc.pool.Dump(ctx)
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
			Timestamp:         ts,

			// TODO: Generate SignatureScript
			OutputScript: prev.OutputScript,
		},
	}

	tree, err := fc.store.StateTree(ctx, prev.Height)
	if err != nil {
		return nil, nil, errors.Wrap(err, "loading state tree")
	}

	view := state.NewMemView(tree, nil)
	ctx = span.NewContextSuffix(ctx, "-validate-all")
	defer span.Finish(ctx)
	for _, tx := range txs {
		if validation.ValidateTxInputs(ctx, view, tx) == nil {
			validation.ApplyTx(ctx, view, tx)
			b.Transactions = append(b.Transactions, tx)
		}
	}

	stateRoot, err := view.StateRoot(ctx)
	if err != nil {
		return nil, nil, err
	}
	b.SetStateRoot(stateRoot)
	b.SetTxRoot(validation.CalcMerkleRoot(b.Transactions))

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

	tree, err := fc.store.StateTree(ctx, block.Height-1)
	if err != nil {
		return errors.Wrap(err, "loading state tree")
	}

	mv := state.NewMemView(tree, nil)
	err = fc.validateBlock(ctx, block, mv)
	if err != nil {
		return errors.Wrap(err, "block validation")
	}
	_, err = mv.StateRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "adding circulation to state tree")
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

	err = fc.store.FinalizeBlock(ctx, block.Height)
	if err != nil {
		return errors.Wrap(err, "finalizing block")
	}

	// When fc.store is a txdb.Store, and fc has been initialized with a
	// channel from txdb.ListenBlocks, then the above call to
	// fc.store.FinalizeBlock will have done a postgresql NOTIFY and
	// that will wake up the goroutine in NewFC, which also calls
	// setHeight.  But duplicate calls with the same blockheight are
	// harmless; and the following call is required in the cases where
	// it's not redundant.
	fc.setHeight(block.Height)

	return nil
}

func (fc *FC) setHeight(h uint64) {
	// We call setHeight from two places independently:
	// ApplyBlock and the Postgres LISTEN goroutine.
	// This means we can get here twice for each block,
	// and any of them might be arbitrarily delayed,
	// which means h might be from the past.
	// Detect and discard these duplicate calls.

	fc.height.cond.L.Lock()
	defer fc.height.cond.L.Unlock()

	if h <= fc.height.n {
		return
	}
	fc.height.n = h
	fc.height.cond.Broadcast()
}

// ValidateBlockForSig performs validation on an incoming _unsigned_
// block in preparation for signing it.  By definition it does not
// execute the sigscript.
func (fc *FC) ValidateBlockForSig(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	prevBlock, err := fc.LatestBlock(ctx)
	if err != nil && errors.Root(err) != ErrNoBlocks {
		return errors.Wrap(err, "getting latest known block")
	}

	tree, err := fc.store.StateTree(ctx, prevBlock.Height)
	if err != nil {
		return errors.Wrap(err, "loading state tree")
	}

	mv := state.NewMemView(tree, nil)
	err = validation.ValidateBlockForSig(ctx, mv, prevBlock, block)
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

func (fc *FC) applyBlock(
	ctx context.Context,
	block *bc.Block,
	mv *state.MemView,
) (newTxs []*bc.Tx, conflictingTxs []*bc.Tx, err error) {
	added := make([]*state.Output, 0, len(mv.Added))
	for _, out := range mv.Added {
		added = append(added, out)
	}
	consumed := make([]*state.Output, 0, len(mv.Consumed))
	for _, out := range mv.Consumed {
		consumed = append(consumed, out)
	}

	newTxs, err = fc.store.ApplyBlock(ctx, block, added, consumed, mv.Assets, mv.StateTree)
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

	txs, err := fc.pool.Dump(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	tree, err := fc.store.StateTree(ctx, block.Height)
	if err != nil {
		return nil, errors.Wrap(err, "loading state tree")
	}

	view := state.NewMemView(tree, nil)
	for _, tx := range txs {
		for _, out := range tx.Outputs {
			if _, ok := view.Assets[out.AssetID]; !ok {
				view.Assets[out.AssetID] = &state.AssetState{}
			}
		}

		// Have to explicitly check that tx is not in block
		// because issuance transactions are always valid, even duplicates.
		// TODO(erykwalder): Remove this check when issuances become unique
		txErr := validation.ValidateTxInputs(ctx, view, tx)
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
				if !view.IsUTXO(ctx, state.Prevout(in)) {
					log.Messagef(ctx, "conflict tx %v missing or spent input %d (%v)", tx.Hash, i, in.Previous)
				}
			}
		}
	}

	err = fc.pool.Clean(ctx, confirmedTxs, conflictTxs, view.Assets)
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
