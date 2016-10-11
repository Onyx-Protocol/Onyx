package core

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"chain/core/account"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/net/http/reqid"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	defaultTxTTL = 5 * time.Minute
)

func buildSingle(ctx context.Context, req *buildRequest) (*txbuilder.Template, error) {
	err := filterAliases(ctx, req)
	if err != nil {
		return nil, err
	}
	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for _, act := range req.Actions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "no action type provided")
		}
		decoder, ok := actionDecoders[typ]
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "unknown action type %q", typ)
		}

		// Remarshal to JSON, the action may have been modified when we
		// filtered aliases.
		b, err := json.Marshal(act)
		if err != nil {
			return nil, err
		}
		a, err := decoder(b)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}

	tpl, err := txbuilder.Build(ctx, req.Tx, actions)
	if err != nil {
		return nil, err
	}

	// ensure null is never returned for signing instructions
	if tpl.SigningInstructions == nil {
		tpl.SigningInstructions = []*txbuilder.SigningInstruction{}
	}
	return tpl, nil
}

// POST /build-transaction
func build(ctx context.Context, buildReqs []*buildRequest) (interface{}, error) {
	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()

			resp, err := buildSingle(reqid.NewSubContext(ctx, reqid.New()), buildReqs[i])
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

func submitSingle(ctx context.Context, c *protocol.Chain, x submitSingleArg) (interface{}, error) {
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

	err := finalizeTxWait(ctx, c, x.tpl)
	if err != nil {
		return nil, err
	}

	return map[string]string{"id": x.tpl.Transaction.Hash().String()}, nil
}

// recordSubmittedTx records a lower bound height at which the tx
// was first submitted to the tx pool. If this request fails for
// some reason, a retry will know to look for the transaction in
// blocks starting at this height.
//
// If the tx has already been submitted, it returns the existing
// height.
// TODO(jackson): Prune entries older than some threshold periodically.
func recordSubmittedTx(ctx context.Context, txHash bc.Hash, currentHeight uint64) (height uint64, err error) {
	const q = `
		WITH inserted AS (
			INSERT INTO submitted_txs (tx_id, height) VALUES($1, $2)
			ON CONFLICT DO NOTHING RETURNING height
		)
		SELECT height FROM inserted
		UNION
		SELECT height FROM submitted_txs WHERE tx_id = $1
	`
	err = pg.QueryRow(ctx, q, txHash, currentHeight).Scan(&height)
	return height, err
}

// CleanupSubmittedTxs will periodically delete records of submitted txs
// older than a day. This function blocks and only exits when its context
// is cancelled.
// TODO(jackson): unexport this and start it in a goroutine in a core.New()
// function?
func CleanupSubmittedTxs(ctx context.Context, db pg.DB) {
	ticker := time.NewTicker(15 * time.Minute)
	for {
		select {
		case <-ticker.C:
			// TODO(jackson): We could avoid expensive bulk deletes by partitioning
			// the table and DROP-ing tables of expired rows. Partitioning doesn't
			// play well with ON CONFLICT clauses though, so we would need to rework
			// how we guarantee uniqueness.
			const q = `DELETE FROM submitted_txs WHERE submitted_at < now() - interval '1 day'`
			_, err := db.Exec(ctx, q)
			if err != nil {
				log.Error(ctx, err)
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

// finalizeTxWait calls FinalizeTx and then waits for confirmation of
// the transaction.  A nil error return means the transaction is
// confirmed on the blockchain.  ErrRejected means a conflicting tx is
// on the blockchain.  context.DeadlineExceeded means ctx is an
// expiring context that timed out.
func finalizeTxWait(ctx context.Context, c *protocol.Chain, txTemplate *txbuilder.Template) error {
	if txTemplate.Transaction == nil {
		return errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	// Avoid a race condition.  Calling c.Height() here ensures that
	// when we start waiting for blocks below, we don't begin waiting at
	// block N+1 when the tx we want is in block N.
	tx := bc.NewTx(*txTemplate.Transaction)
	height, err := recordSubmittedTx(ctx, tx.Hash, c.Height())
	if err != nil {
		return errors.Wrap(err, "saving tx submitted height")
	}

	err = txbuilder.FinalizeTx(ctx, c, tx)
	if err != nil {
		return err
	}

	// As a rule we only index confirmed blockchain data to prevent dirty
	// reads, but here we're explicitly breaking that rule iff all of the
	// inputs to the transaction are from locally-controlled keys. In that
	// case, we're confident that this tx will be confirmed, so we relax
	// that constraint to allow use of unconfirmed change, etc.
	if txTemplate.Local {
		err := account.IndexUnconfirmedUTXOs(ctx, tx)
		if err != nil {
			return errors.Wrap(err, "indexing unconfirmed account utxos")
		}
	}

	for {
		height++
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-waitBlock(ctx, c, height):
			b, err := c.GetBlock(ctx, height)
			if err != nil {
				return errors.Wrap(err, "getting block that just landed")
			}
			for _, confirmed := range b.Transactions {
				if confirmed.Hash == tx.Hash {
					// confirmed
					return nil
				}
			}

			if tx.MaxTime > 0 && tx.MaxTime < b.TimestampMS {
				return txbuilder.ErrRejected
			}

			// might still be in pool or might be rejected; we can't
			// tell definitively until its max time elapses.

			// Re-insert into the pool in case it was dropped.
			err = txbuilder.FinalizeTx(ctx, c, tx)
			if err != nil {
				return err
			}

			// TODO(jackson): Do simple rejection checks like checking if
			// the tx's blockchain prevouts still exist in the state tree.
		}
	}
}

func waitBlock(ctx context.Context, c *protocol.Chain, height uint64) <-chan struct{} {
	done := make(chan struct{}, 1)
	go func() {
		c.WaitForBlock(height)
		done <- struct{}{}
	}()
	return done
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
func (h *Handler) submit(ctx context.Context, x submitArg) interface{} {
	responses := make([]interface{}, len(x.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			resp, err := submitSingle(reqid.NewSubContext(ctx, reqid.New()), h.Chain, submitSingleArg{tpl: x.Transactions[i], wait: x.wait})
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
