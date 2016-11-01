package core

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"chain/core/fetch"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/log"
	"chain/net/http/reqid"
	"chain/protocol"
	"chain/protocol/bc"
)

var defaultTxTTL = 5 * time.Minute

func (h *Handler) buildSingle(ctx context.Context, req *buildRequest) (*txbuilder.Template, error) {
	err := h.filterAliases(ctx, req)
	if err != nil {
		return nil, err
	}
	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for i, act := range req.Actions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "no action type provided on action %d", i)
		}
		decoder, ok := h.actionDecoders[typ]
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "unknown action type %q on action %d", typ, i)
		}

		// Remarshal to JSON, the action may have been modified when we
		// filtered aliases.
		b, err := json.Marshal(act)
		if err != nil {
			return nil, err
		}
		a, err := decoder(b)
		if err != nil {
			return nil, errors.WithDetailf(errBadAction, "%s on action %d", err.Error(), i)
		}
		actions = append(actions, a)
	}

	ttl := req.TTL.Duration
	if ttl == 0 {
		ttl = defaultTxTTL
	}
	maxTime := time.Now().Add(ttl)
	tpl, err := txbuilder.Build(ctx, req.Tx, actions, maxTime)
	if errors.Root(err) == txbuilder.ErrAction {
		err = errors.WithData(err, "actions", errInfoBodyList(errors.Data(err)["actions"].([]error)))
	}
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
func (h *Handler) build(ctx context.Context, buildReqs []*buildRequest) (interface{}, error) {
	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()

			resp, err := h.buildSingle(reqid.NewSubContext(ctx, reqid.New()), buildReqs[i])
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
	wait chainjson.Duration
}

func (h *Handler) submitSingle(ctx context.Context, x submitSingleArg) (interface{}, error) {
	// TODO(bobg): Set up an expiring context object outside this
	// function, perhaps in handler.ServeHTTPContext, and perhaps
	// initialize the timeout from the HTTP Timeout field.  (Or just
	// switch to gRPC.)
	timeout := x.wait.Duration
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := h.finalizeTxWait(ctx, x.tpl)
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
func recordSubmittedTx(ctx context.Context, db pg.DB, txHash bc.Hash, currentHeight uint64) (height uint64, err error) {
	const q = `
		WITH inserted AS (
			INSERT INTO submitted_txs (tx_id, height) VALUES($1, $2)
			ON CONFLICT DO NOTHING RETURNING height
		)
		SELECT height FROM inserted
		UNION
		SELECT height FROM submitted_txs WHERE tx_id = $1
	`
	err = db.QueryRow(ctx, q, txHash, currentHeight).Scan(&height)
	return height, err
}

// CleanupSubmittedTxs will periodically delete records of submitted txs
// older than a day. This function blocks and only exits when its context
// is cancelled.
//
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
func (h *Handler) finalizeTxWait(ctx context.Context, txTemplate *txbuilder.Template) error {
	if txTemplate.Transaction == nil {
		return errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	// Use the current generator height as the lower bound of the block height
	// that the transaction may appear in.
	generatorHeight, _ := fetch.GeneratorHeight()
	localHeight := h.Chain.Height()
	if localHeight > generatorHeight {
		generatorHeight = localHeight
	}

	// Remember this height in case we retry this submit call.
	tx := bc.NewTx(*txTemplate.Transaction)
	height, err := recordSubmittedTx(ctx, h.DB, tx.Hash, generatorHeight)
	if err != nil {
		return errors.Wrap(err, "saving tx submitted height")
	}

	err = txbuilder.FinalizeTx(ctx, h.Chain, tx)
	if err != nil {
		return err
	}

	// As a rule we only index confirmed blockchain data to prevent dirty
	// reads, but here we're explicitly breaking that rule iff all of the
	// inputs to the transaction are from locally-controlled keys. In that
	// case, we're confident that this tx will be confirmed, so we relax
	// that constraint to allow use of unconfirmed change, etc.
	if txTemplate.Local {
		err := h.Accounts.IndexUnconfirmedUTXOs(ctx, tx)
		if err != nil {
			return errors.Wrap(err, "indexing unconfirmed account utxos")
		}
	}

	height, err = waitForTxInBlock(ctx, h.Chain, tx, height)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.PinStore.Pin(query.TxPinName).WaitForHeight(height):
	}

	return nil
}

func waitForTxInBlock(ctx context.Context, c *protocol.Chain, tx *bc.Tx, height uint64) (uint64, error) {
	for {
		height++
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		case <-c.WaitForBlock(height):
			b, err := c.GetBlock(ctx, height)
			if err != nil {
				return 0, errors.Wrap(err, "getting block that just landed")
			}
			for _, confirmed := range b.Transactions {
				if confirmed.Hash == tx.Hash {
					// confirmed
					return height, nil
				}
			}

			if tx.MaxTime > 0 && tx.MaxTime < b.TimestampMS {
				return 0, txbuilder.ErrRejected
			}

			// might still be in pool or might be rejected; we can't
			// tell definitively until its max time elapses.

			// Re-insert into the pool in case it was dropped.
			err = txbuilder.FinalizeTx(ctx, c, tx)
			if err != nil {
				return 0, err
			}

			// TODO(jackson): Do simple rejection checks like checking if
			// the tx's blockchain prevouts still exist in the state tree.
		}
	}
}

type submitArg struct {
	Transactions []*txbuilder.Template
	wait         chainjson.Duration
}

// POST /v3/transact/submit
// Idempotent
func (h *Handler) submit(ctx context.Context, x submitArg) interface{} {
	responses := make([]interface{}, len(x.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			resp, err := h.submitSingle(reqid.NewSubContext(ctx, reqid.New()), submitSingleArg{tpl: x.Transactions[i], wait: x.wait})
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
