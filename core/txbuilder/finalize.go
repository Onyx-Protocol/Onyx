package txbuilder

import (
	"bytes"
	"context"

	"chain/core/rpc"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc/legacy"
	"chain/protocol/vm"
)

var (
	// ErrRejected means the network rejected a tx (as a double-spend)
	ErrRejected = errors.New("transaction rejected")

	ErrMissingRawTx        = errors.New("missing raw tx")
	ErrBadInstructionCount = errors.New("too many signing instructions in template")
)

// Submitter submits a transaction to the generator so that it may
// be confirmed in a block.
type Submitter interface {
	Submit(ctx context.Context, tx *legacy.Tx) error
}

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, c *protocol.Chain, s Submitter, tx *legacy.Tx) error {
	err := checkTxSighashCommitment(tx)
	if err != nil {
		return err
	}

	// Make sure there is at least one block in case client is trying to
	// finalize a tx before the initial block has landed
	<-c.BlockWaiter(1)

	err = c.ValidateTx(tx.Tx)
	if errors.Root(err) == protocol.ErrBadTx {
		return errors.Sub(ErrRejected, err)
	}
	if err != nil {
		return errors.Wrap(err, "tx rejected")
	}
	if tx.Tx.MaxTimeMs > 0 && tx.Tx.MaxTimeMs < c.TimestampMS() {
		return errors.Wrap(ErrRejected, "tx expired")
	}

	err = s.Submit(ctx, tx)
	return errors.Wrap(err)
}

var (
	// ErrNoTxSighashCommitment is returned when no input commits to the
	// complete transaction.
	// To permit idempotence of transaction submission, we require at
	// least one input to commit to the complete transaction (what you get
	// when you build a transaction with allow_additional_actions=false).
	ErrNoTxSighashCommitment = errors.New("no commitment to tx sighash")

	// ErrNoTxSighashAttempt is returned when there was no attempt made to sign
	// this transaction.
	ErrNoTxSighashAttempt = errors.New("no tx sighash attempted")

	// ErrTxSignatureFailure is returned when there was an attempt to sign this
	// transaction, but it failed.
	ErrTxSignatureFailure = errors.New("tx signature was attempted but failed")
)

func checkTxSighashCommitment(tx *legacy.Tx) error {
	var lastError error

	for i, inp := range tx.Inputs {
		var args [][]byte
		switch t := inp.TypedInput.(type) {
		case *legacy.SpendInput:
			args = t.Arguments
		case *legacy.IssuanceInput:
			args = t.Arguments
		}
		// Note: These numbers will need to change if more args are added such that the minimum length changes
		switch {
		// A conforming arguments list contains
		// [... arg1 arg2 ... argN N sig1 sig2 ... sigM prog]
		// The args are the opaque arguments to prog. In the case where
		// N is 0 (prog takes no args), and assuming there must be at
		// least one signature, args has a minimum length of 3.
		case len(args) == 0:
			lastError = ErrNoTxSighashAttempt
			continue
		case len(args) < 3:
			lastError = ErrTxSignatureFailure
			continue
		}
		lastError = ErrNoTxSighashCommitment
		prog := args[len(args)-1]
		if len(prog) != 35 {
			continue
		}
		if prog[0] != byte(vm.OP_DATA_32) {
			continue
		}
		if !bytes.Equal(prog[33:], []byte{byte(vm.OP_TXSIGHASH), byte(vm.OP_EQUAL)}) {
			continue
		}
		h := tx.SigHash(uint32(i))
		if !bytes.Equal(h.Bytes(), prog[1:33]) {
			continue
		}
		// At least one input passes commitment checks
		return nil
	}

	return lastError
}

// RemoteGenerator implements the Submitter interface and submits the
// transaction to a remote generator.
// TODO(jackson): This implementation maybe belongs elsewhere.
type RemoteGenerator struct {
	Peer *rpc.Client
}

func (rg *RemoteGenerator) Submit(ctx context.Context, tx *legacy.Tx) error {
	err := rg.Peer.Call(ctx, "/rpc/submit", tx, nil)
	err = errors.Wrap(err, "generator transaction notice")
	return err
}
