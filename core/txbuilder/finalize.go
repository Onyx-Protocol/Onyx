package txbuilder

import (
	"bytes"
	"context"

	"chain/core/pb"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/validation"
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
	Submit(ctx context.Context, tx *bc.Tx) error
}

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, c *protocol.Chain, s Submitter, tx *bc.Tx) error {
	err := checkTxSighashCommitment(tx)
	if err != nil {
		return err
	}

	// Make sure there is at least one block in case client is trying to
	// finalize a tx before the initial block has landed
	<-c.BlockWaiter(1)

	// If this transaction is valid, ValidateTxCached will store it in the cache.
	err = c.ValidateTxCached(tx)
	if errors.Root(err) == validation.ErrBadTx {
		detail := errors.Detail(err)
		err = errors.Wrap(ErrRejected, err)
		return errors.WithDetail(err, detail)
	} else if err != nil {
		return errors.Wrap(err, "tx rejected")
	}

	err = s.Submit(ctx, tx)
	return errors.Wrap(err)
}

// To permit idempotence of transaction submission, we require at
// least one input to commit to the complete transaction (what you get
// when you build a transaction with allow_additional_actions=false).
var ErrNoTxSighashCommitment = errors.New("no commitment to tx sighash")

func checkTxSighashCommitment(tx *bc.Tx) error {
	allIssuances := true
	sigHasher := bc.NewSigHasher(&tx.TxData)

	for i, inp := range tx.Inputs {
		var args [][]byte
		switch t := inp.TypedInput.(type) {
		case *bc.SpendInput:
			args = t.Arguments
			allIssuances = false
		case *bc.IssuanceInput:
			args = t.Arguments
		}
		if len(args) < 3 {
			// A conforming arguments list contains
			// [... arg1 arg2 ... argN N sig1 sig2 ... sigM prog]
			// The args are the opaque arguments to prog. In the case where
			// N is 0 (prog takes no args), and assuming there must be at
			// least one signature, args has a minimum length of 3.
			continue
		}
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
		h := sigHasher.Hash(uint32(i))
		if !bytes.Equal(h[:], prog[1:33]) {
			continue
		}
		return nil
	}

	if !allIssuances {
		return ErrNoTxSighashCommitment
	}

	return nil
}

// RemoteGenerator implements the Submitter interface and submits the
// transaction to a remote generator.
// TODO(jackson): This implementation maybe belongs elsewhere.
type RemoteGenerator struct {
	Peer pb.NodeClient
}

func (rg *RemoteGenerator) Submit(ctx context.Context, tx *bc.Tx) error {
	var buf bytes.Buffer
	_, err := tx.WriteTo(&buf)
	if err != nil {
		return errors.Wrap(err, "couldn't write tx")
	}

	_, err = rg.Peer.SubmitTx(ctx, &pb.SubmitTxRequest{Transaction: buf.Bytes()})
	if err != nil {
		err = errors.Wrap(err, "generator transaction notice")
		log.Error(ctx, err)

		// Return an error so that the client knows that it needs to
		// retry the request.
		return err
	}
	return nil
}
