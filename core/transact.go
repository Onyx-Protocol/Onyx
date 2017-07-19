package core

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"chain/core/leader"
	"chain/core/txbuilder"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/log"
	"chain/net/http/httperror"
	"chain/net/http/reqid"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

const defaultTxTTL = 5 * time.Minute

func (a *API) actionDecoder(action string) (func([]byte) (txbuilder.Action, error), bool) {
	var decoder func([]byte) (txbuilder.Action, error)
	switch action {
	case "control_account":
		decoder = a.accounts.DecodeControlAction
	case "control_program":
		decoder = txbuilder.DecodeControlProgramAction
	case "control_receiver":
		decoder = txbuilder.DecodeControlReceiverAction
	case "issue":
		decoder = a.assets.DecodeIssueAction
	case "retire":
		decoder = txbuilder.DecodeRetireAction
	case "spend_account":
		decoder = a.accounts.DecodeSpendAction
	case "spend_account_unspent_output":
		decoder = a.accounts.DecodeSpendUTXOAction
	case "set_transaction_reference_data":
		decoder = txbuilder.DecodeSetTxRefDataAction
	default:
		return nil, false
	}
	return decoder, true
}

func (a *API) buildSingle(ctx context.Context, req *buildRequest) (*txbuilder.Template, error) {
	err := a.filterAliases(ctx, req)
	if err != nil {
		return nil, err
	}
	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for i, act := range req.Actions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "no action type provided on action %d", i)
		}
		decoder, ok := a.actionDecoder(typ)
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
		// Format each of the inner errors contained in the data.
		var formattedErrs []httperror.Response
		for _, innerErr := range errors.Data(err)["actions"].([]error) {
			resp := errorFormatter.Format(innerErr)
			formattedErrs = append(formattedErrs, resp)
		}
		err = errors.WithData(err, "actions", formattedErrs)
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
func (a *API) build(ctx context.Context, buildReqs []*buildRequest) (interface{}, error) {
	// If we're not the leader, we don't have access to the current
	// reservations. Forward the build call to the leader process.
	// TODO(jackson): Distribute reservations across cored processes.
	if a.leader.State() != leader.Leading {
		var resp interface{}
		err := a.forwardToLeader(ctx, "/build-transaction", buildReqs, &resp)
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

			tmpl, err := a.buildSingle(subctx, buildReqs[i])
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

func (a *API) submitSingle(ctx context.Context, tpl *txbuilder.Template, waitUntil string) (interface{}, error) {
	if tpl.Transaction == nil {
		return nil, errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	err := a.finalizeTxWait(ctx, tpl, waitUntil)
	if err != nil {
		return nil, errors.Wrapf(err, "tx %s", tpl.Transaction.ID.String())
	}

	return map[string]string{"id": tpl.Transaction.ID.String()}, nil
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
	res, err := db.ExecContext(ctx, insertQ, txHash.Bytes(), currentHeight)
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
	err = db.QueryRowContext(ctx, selectQ, txHash.Bytes()).Scan(&height)
	return height, err
}

// cleanUpSubmittedTxs will periodically delete records of submitted txs
// older than a day. This function blocks and only exits when its context
// is cancelled.
func cleanUpSubmittedTxs(ctx context.Context, db pg.DB) {
	ticker := time.NewTicker(15 * time.Minute)
	for {
		select {
		case <-ticker.C:
			// TODO(jackson): We could avoid expensive bulk deletes by partitioning
			// the table and DROP-ing tables of expired rows. Partitioning doesn't
			// play well with ON CONFLICT clauses though, so we would need to rework
			// how we guarantee uniqueness.
			const q = `DELETE FROM submitted_txs WHERE submitted_at < now() - interval '1 day'`
			_, err := db.ExecContext(ctx, q)
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
func (a *API) finalizeTxWait(ctx context.Context, txTemplate *txbuilder.Template, waitUntil string) error {
	// Use the current generator height as the lower bound of the block height
	// that the transaction may appear in.
	var generatorHeight uint64
	if a.replicator != nil {
		generatorHeight, _ = a.replicator.PeerHeight()
	}
	localHeight := a.chain.Height()
	if localHeight > generatorHeight {
		generatorHeight = localHeight
	}

	// Remember this height in case we retry this submit call.
	height, err := recordSubmittedTx(ctx, a.db, txTemplate.Transaction.ID, generatorHeight)
	if err != nil {
		return errors.Wrap(err, "saving tx submitted height")
	}

	err = txbuilder.FinalizeTx(ctx, a.chain, a.submitter, txTemplate.Transaction)
	if err != nil {
		return err
	}
	if waitUntil == "none" {
		return nil
	}

	height, err = a.waitForTxInBlock(ctx, txTemplate.Transaction, height)
	if err != nil {
		return err
	}
	if waitUntil == "confirmed" {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.pinStore.AllWaiter(height):
	}

	return nil
}

func (a *API) waitForTxInBlock(ctx context.Context, tx *legacy.Tx, height uint64) (uint64, error) {
	for {
		height++
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		case <-a.chain.BlockWaiter(height):
			b, err := a.chain.GetBlock(ctx, height)
			if err != nil {
				return 0, errors.Wrap(err, "getting block that just landed")
			}
			for _, confirmed := range b.Transactions {
				if confirmed.ID == tx.ID {
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
			err = txbuilder.FinalizeTx(ctx, a.chain, a.submitter, tx)
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
func (a *API) submit(ctx context.Context, x submitArg) (interface{}, error) {
	if a.leader.State() != leader.Leading {
		var resp json.RawMessage
		err := a.forwardToLeader(ctx, "/submit-transaction", x, &resp)
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

			tx, err := a.submitSingle(subctx, &x.Transactions[i], x.WaitUntil)
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
