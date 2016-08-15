package cos

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/cos/validation"
	"chain/crypto/ed25519"
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

	timestampMS := bc.Millis(now)

	// TODO(jackson): The leader process should store its own in-memory
	// copy of the current state tree and latest block on FC instead.
	prev, err = fc.LatestBlock(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fetch latest block")
	}

	if timestampMS < prev.TimestampMS {
		return nil, nil, fmt.Errorf("timestamp %d is earlier than prevblock timestamp %d", timestampMS, prev.TimestampMS)
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
			TimestampMS:       timestampMS,
			OutputScript:      prev.OutputScript,
		},
	}

	snapshot, err := fc.currentState(ctx, prev.Height)
	if err != nil {
		return nil, nil, err
	}

	ctx = span.NewContextSuffix(ctx, "-validate-all")
	defer span.Finish(ctx)
	for _, tx := range txs {
		if validation.ConfirmTx(snapshot, tx, b.TimestampMS) == nil {
			validation.ApplyTx(snapshot, tx)
			b.Transactions = append(b.Transactions, tx)
		}
	}

	b.SetStateRoot(snapshot.Tree.RootHash())
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

	snapshot, err := fc.currentState(ctx, block.Height-1)
	if err != nil {
		return err
	}

	err = fc.validateBlock(ctx, block, snapshot)
	if err != nil {
		return errors.Wrap(err, "block validation")
	}

	_, err = fc.applyBlock(ctx, block, snapshot)
	if err != nil {
		return errors.Wrap(err, "applying block")
	}

	for _, tx := range block.Transactions {
		for _, cb := range fc.txCallbacks {
			cb(ctx, tx)
		}
	}

	for _, cb := range fc.blockCallbacks {
		cb(ctx, block)
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

func (fc *FC) currentState(ctx context.Context, expectedHeight uint64) (*state.Snapshot, error) {
	// TODO(jackson): Store the state tree on FC.
	snapshot, height, err := fc.store.LatestSnapshot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "loading state snapshot")
	}
	if height != expectedHeight {
		return nil, errors.New("missing state snapshot for block")
	}
	return state.Copy(snapshot), nil
}

// ValidateBlockForSig performs validation on an incoming _unsigned_
// block in preparation for signing it.  By definition it does not
// execute the sigscript.
func (fc *FC) ValidateBlockForSig(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		prev     *bc.Block
		snapshot = state.Empty()
	)

	if block.Height > 1 {
		var err error
		prev, err = fc.store.GetBlock(ctx, block.Height-1)
		if err != nil {
			return errors.Wrap(err, "getting previous block")
		}

		snapshot, err = fc.currentState(ctx, prev.Height)
		if err != nil {
			return err
		}
	}

	err := validation.ValidateBlockForSig(ctx, snapshot, prev, block)
	return errors.Wrap(err, "validation")
}

// validateBlock performs validation on an incoming block, in advance of
// applying the block to the store.
func (fc *FC) validateBlock(ctx context.Context, block *bc.Block, snapshot *state.Snapshot) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	prev, err := fc.store.GetBlock(ctx, block.Height-1)
	if err != nil {
		return errors.Wrap(err, "loading previous block")
	}
	err = validation.ValidateBlockHeader(prev, block)
	if err != nil {
		return errors.Wrap(err, "validating block header")
	}

	if isSignedByTrustedHost(block, fc.trustedKeys) {
		err = validation.ApplyBlock(snapshot, block)
	} else {
		err = validation.ValidateAndApplyBlock(ctx, snapshot, prev, block)
	}
	if err != nil {
		return errors.Wrapf(ErrBadBlock, "validate block: %v", err)
	}
	return nil
}

func isSignedByTrustedHost(block *bc.Block, trustedKeys []ed25519.PublicKey) bool {
	sigs, err := txscript.PushedData(block.SignatureScript)
	if err != nil {
		return false
	}

	hash := block.HashForSig()
	for _, sig := range sigs {
		if len(sig) == 0 {
			continue
		}
		for _, pubk := range trustedKeys {
			if ed25519.Verify(pubk, hash[:], sig) {
				return true
			}
		}
	}

	return false
}

func (fc *FC) applyBlock(
	ctx context.Context,
	block *bc.Block,
	snapshot *state.Snapshot,
) (conflictingTxs []*bc.Tx, err error) {
	err = fc.store.SaveBlock(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "storing block")
	}
	err = fc.store.SaveSnapshot(ctx, block.Height, snapshot)
	if err != nil {
		return nil, errors.Wrap(err, "storing state snapshot")
	}
	conflicts, err := fc.rebuildPool(ctx, block)
	return conflicts, errors.Wrap(err, "rebuilding pool")
}

func (fc *FC) rebuildPool(ctx context.Context, block *bc.Block) ([]*bc.Tx, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	txInBlock := make(map[bc.Hash]bool)
	for _, tx := range block.Transactions {
		txInBlock[tx.Hash] = true
	}

	var (
		deleteTxs   []*bc.Tx
		conflictTxs []*bc.Tx
	)

	txs, err := fc.pool.Dump(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	snapshot, err := fc.currentState(ctx, block.Height)
	if err != nil {
		return nil, err
	}

	for _, tx := range txs {
		txErr := validation.ConfirmTx(snapshot, tx, block.TimestampMS)
		if txErr == nil {
			validation.ApplyTx(snapshot, tx)
		} else {
			deleteTxs = append(deleteTxs, tx)
			if txInBlock[tx.Hash] {
				continue
			}

			// This should never happen in sandbox, unless a reservation expired
			// before the original tx was finalized.
			log.Messagef(ctx, "deleting conflict tx %v because %q", tx.Hash, txErr)
			conflictTxs = append(conflictTxs, tx)
		}
	}

	err = fc.pool.Clean(ctx, deleteTxs)
	if err != nil {
		return nil, errors.Wrap(err, "removing conflicting txs")
	}

	return conflictTxs, nil
}

// ComputeBlockSignature signs a block with the given key.  It does
// not validate the block.
func ComputeBlockSignature(b *bc.Block, key ed25519.PrivateKey) []byte {
	hash := b.HashForSig()
	return ed25519.Sign(key, hash[:])
}

// AddSignaturesToBlock adds signatures to a block, replacing the
// block's SignatureScript.  The signatures must be in the correct
// order, to wit: matching the order of pubkeys in the previous
// block's output script.
func AddSignaturesToBlock(b *bc.Block, signatures [][]byte) error {
	// assumes multisig output script
	builder := txscript.NewScriptBuilder()
	for _, signature := range signatures {
		builder.AddData(signature)
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
func GenerateBlockScript(keys []ed25519.PublicKey, nSigs int) ([]byte, error) {
	return txscript.BlockMultiSigScript(keys, nSigs)
}

// UpsertGenesisBlock creates a genesis block iff it does not exist.
func (fc *FC) UpsertGenesisBlock(ctx context.Context, pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*bc.Block, error) {
	// TODO(bobg): Cache the genesis block if it exists and return it
	// rather than always consing up a new one.
	b, err := NewGenesisBlock(pubkeys, nSigs, timestamp)
	if err != nil {
		return nil, err
	}

	h, err := fc.store.Height(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting blockchain height")
	}
	if h == 0 {
		err = fc.AddBlock(ctx, b)
		if err != nil {
			return nil, errors.Wrap(err, "adding genesis block")
		}
	}
	return b, nil
}

func NewGenesisBlock(pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*bc.Block, error) {
	script, err := GenerateBlockScript(pubkeys, nSigs)
	if err != nil {
		return nil, err
	}
	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:      bc.NewBlockVersion,
			Height:       1,
			TimestampMS:  bc.Millis(timestamp),
			OutputScript: script,
		},
	}
	return b, nil
}
