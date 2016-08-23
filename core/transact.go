package core

import (
	"context"
	"sync"
	"time"

	"chain/core/account"
	"chain/core/txbuilder"
	"chain/cos"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/reqid"
	"chain/net/trace/span"
)

func buildSingle(ctx context.Context, req *buildRequest) (*txbuilder.Template, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback(ctx)

	tpl, err := txbuilder.Build(ctx, req.Tx, req.actions(), req.ReferenceData)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	// ensure null is never returned for inputs
	if tpl.Inputs == nil {
		tpl.Inputs = []*txbuilder.Input{}
	}

	return tpl, nil
}

// POST /build-transaction-template
func build(ctx context.Context, buildReqs []*aliasBuildRequest) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()

			filteredRequest, err := filterAliases(ctx, buildReqs[i])
			if err != nil {
				logHTTPError(ctx, err)
				responses[i], _ = errInfo(err)
				return
			}

			resp, err := buildSingle(reqid.NewSubContext(ctx, reqid.New()), filteredRequest)
			if err != nil {
				logHTTPError(ctx, err)
				responses[i], _ = errInfo(err)
			} else {
				responses[i] = resp
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}

type submitSingleArg struct {
	tpl  *txbuilder.Template
	wait time.Duration
}

func submitSingle(ctx context.Context, fc *cos.FC, x submitSingleArg) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	// TODO(bobg): Set up an expiring context object outside this
	// function, perhaps in handler.ServeHTTPContext, and perhaps
	// initialize the timeout from the HTTP Timeout field.  (Or just
	// switch to gRPC.)
	timeout := x.wait
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tx, err := finalizeTxWait(ctx, fc, x.tpl)
	if err != nil {
		return nil, err
	}

	return map[string]string{"id": tx.Hash.String()}, nil
}

// finalizeTxWait calls FinalizeTx and then waits for confirmation of
// the transaction.  A nil error return means the transaction is
// confirmed on the blockchain.  ErrRejected means a conflicting tx is
// on the blockchain.  context.DeadlineExceeded means ctx is an
// expiring context that timed out.
func finalizeTxWait(ctx context.Context, fc *cos.FC, txTemplate *txbuilder.Template) (*bc.Tx, error) {
	// Avoid a race condition.  Calling fc.Height() here ensures that
	// when we start waiting for blocks below, we don't begin waiting at
	// block N+1 when the tx we want is in block N.
	height := fc.Height()

	tx, err := txbuilder.FinalizeTx(ctx, fc, txTemplate)
	if err != nil {
		return nil, err
	}

	// As a rule we only index confirmed blockchain data to prevent dirty
	// reads, but here we're explicitly breaking that rule iff all of the
	// inputs to the transaction are from locally-controlled keys. In that
	// case, we're confident that this tx will be confirmed, so we relax
	// that constraint to allow use of unconfirmed change, etc.
	if txTemplate.Local {
		err := account.IndexUnconfirmedUTXOs(ctx, tx)
		if err != nil {
			return nil, errors.Wrap(err, "indexing unconfirmed account utxos")
		}
	}

	for {
		height++
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case err := <-waitBlock(ctx, fc, height):
			if err != nil {
				// This should be impossible, since the only error produced by
				// WaitForBlock is ErrTheDistantFuture, and height is known
				// not to be in "the distant future."
				return nil, errors.Wrapf(err, "waiting for block %d", height)
			}
			// TODO(bobg): This technique is not future-proof.  The database
			// won't necessarily contain all the txs we might care about.
			// An alternative approach will be to scan through each block as
			// it lands, looking for the tx or a tx that conflicts with it.
			// For now, though, this is probably faster and simpler.
			bcTxs, err := fc.ConfirmedTxs(ctx, tx.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "getting bc txs")
			}
			if _, ok := bcTxs[tx.Hash]; ok {
				// confirmed
				return tx, nil
			}

			poolTxs, err := fc.PendingTxs(ctx, tx.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "getting pool txs")
			}
			if _, ok := poolTxs[tx.Hash]; !ok {
				// rejected
				return nil, txbuilder.ErrRejected
			}

			// still in the pool; iterate
		}
	}
}

func waitBlock(ctx context.Context, fc *cos.FC, height uint64) <-chan error {
	c := make(chan error, 1)
	go func() { c <- fc.WaitForBlock(ctx, height) }()
	return c
}

// TODO(bobg): allow caller to specify reservation by (encrypted) id?
// POST /v3/assets/cancel-reservation
// Idempotent
func cancelReservation(ctx context.Context, x struct{ Transaction bc.Tx }) error {
	var outpoints []bc.Outpoint
	for _, input := range x.Transaction.Inputs {
		if !input.IsIssuance() {
			outpoints = append(outpoints, input.Outpoint())
		}
	}
	return account.CancelReservations(ctx, outpoints)
}

type submitArg struct {
	Transactions []*txbuilder.Template
	wait         time.Duration
}

// POST /v3/transact/submit
// Idempotent
func (a *api) submit(ctx context.Context, x submitArg) interface{} {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	responses := make([]interface{}, len(x.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			resp, err := submitSingle(reqid.NewSubContext(ctx, reqid.New()), a.fc, submitSingleArg{tpl: x.Transactions[i], wait: x.wait})
			if err != nil {
				logHTTPError(ctx, err)
				responses[i], _ = errInfo(err)
			} else {
				responses[i] = resp
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	return responses
}
