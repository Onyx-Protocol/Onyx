package core

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"chain/core/fetch"
	"chain/core/leader"
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
	// If we're not the leader, we don't have access to the current
	// reservations. Forward the build call to the leader process.
	// TODO(jackson): Distribute reservations across cored processes.
	if !leader.IsLeading() {
		var resp map[string]interface{}
		err := h.forwardToLeader(ctx, "/build-transaction", buildReqs, &resp)
		return resp, err
	}

	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			tmpl, err := h.buildSingle(subctx, buildReqs[i])
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = tmpl
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}

func (h *Handler) submitSingle(ctx context.Context, tpl *txbuilder.Template, waitUntil string) (interface{}, error) {
	err := h.finalizeTxWait(ctx, tpl, waitUntil)
	if err != nil {
		return nil, errors.Wrapf(err, "tx %s", tpl.Transaction.Hash())
	}

	return map[string]string{"id": tpl.Transaction.Hash().String()}, nil
}

// recordSubmittedTx records a lower bound height at which the tx
// was first submitted to the tx pool. If this request fails for
// some reason, a retry will know to look for the transaction in
// blocks starting at this height.
//
// If the tx has already been submitted, it returns the existing
// height.
func recordSubmittedTx(ctx context.Context, db pg.DB, txHash bc.Hash, currentHeight uint64) (uint64, error) {
	const insertQ = `
		INSERT INTO submitted_txs (tx_hash, height) VALUES($1, $2)
		ON CONFLICT DO NOTHING
	`
	res, err := db.Exec(ctx, insertQ, txHash[:], currentHeight)
	if err != nil {
		return 0, err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if inserted == 1 {
		return currentHeight, nil
	}

	// The insert didn't affect any rows, meaning there was already an entry
	// for this transaction hash.
	const selectQ = `
		SELECT height FROM submitted_txs WHERE tx_hash = $1
	`
	var height uint64
	err = db.QueryRow(ctx, selectQ, txHash[:]).Scan(&height)
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
func (h *Handler) finalizeTxWait(ctx context.Context, txTemplate *txbuilder.Template, waitUntil string) error {
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
	if waitUntil == "none" {
		return nil
	}

	height, err = waitForTxInBlock(ctx, h.Chain, tx, height)
	if err != nil {
		return err
	}
	if waitUntil == "confirmed" {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.PinStore.AllWaiter(height):
	}

	return nil
}

func waitForTxInBlock(ctx context.Context, c *protocol.Chain, tx *bc.Tx, height uint64) (uint64, error) {
	for {
		height++
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		case <-c.BlockWaiter(height):
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
				return 0, errors.Wrap(txbuilder.ErrRejected, "transaction max time exceeded")
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
	Transactions []txbuilder.Template
	wait         chainjson.Duration
	WaitUntil    string `json:"wait_until"` // values none, confirmed, processed. default: processed
}

// POST /submit-transaction
func (h *Handler) submit(ctx context.Context, x submitArg) (interface{}, error) {
	if !leader.IsLeading() {
		var resp json.RawMessage
		err := h.forwardToLeader(ctx, "/submit-transaction", x, &resp)
		return resp, err
	}

	// Setup a timeout for the provided wait duration.
	timeout := x.wait.Duration
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	responses := make([]interface{}, len(x.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			tx, err := h.submitSingle(subctx, &x.Transactions[i], x.WaitUntil)
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = tx
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}
