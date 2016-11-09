package txbuilder

import (
	"bytes"
	"context"

	"chain/core/rpc"
	"chain/errors"
	chainlog "chain/log"
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

var Generator *rpc.Client

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, c *protocol.Chain, tx *bc.Tx) error {
	err := publishTx(ctx, c, tx)
	if err != nil {
		rawtx, err2 := tx.MarshalText()
		if err2 != nil {
			// ignore marshalling errors (they should never happen anyway)
			return err
		}
		return errors.Wrapf(err, "tx=%s", rawtx)
	}

	return nil
}

func publishTx(ctx context.Context, c *protocol.Chain, msg *bc.Tx) error {
	err := checkTxSighashCommitment(msg)
	if err != nil {
		return err
	}

	// Make sure there is at least one block in case client is trying to
	// finalize a tx before the initial block has landed
	<-c.BlockWaiter(1)

	if Generator != nil {
		// If this transaction is valid, ValidateTxCached will store it in the cache.
		err := c.ValidateTxCached(msg)
		if err != nil {
			return errors.Wrap(err, "tx rejected")
		}

		err = Generator.Call(ctx, "/rpc/submit", msg, nil)
		if err != nil {
			err = errors.Wrap(err, "generator transaction notice")
			chainlog.Error(ctx, err)

			// Return an error so that the client knows that it needs to
			// retry the request.
			return err
		}
	} else {
		err = c.AddTx(ctx, msg)
		if errors.Root(err) == validation.ErrBadTx {
			detail := errors.Detail(err)
			err = errors.Wrap(ErrRejected, err)
			return errors.WithDetail(err, detail)
		} else if err != nil {
			return errors.Wrap(err, "add tx to blockchain")
		}
	}
	return nil
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
		h := sigHasher.Hash(i)
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
