package core

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/core/fetch"
	"chain/core/leader"
	"chain/core/pb"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
	"chain/net/http/reqid"
	"chain/protocol/bc"
)

var defaultTxTTL = 5 * time.Minute

func (h *Handler) buildSingle(ctx context.Context, req *pb.BuildTxsRequest_Request) (*pb.TxsResponse_Response, error) {
	err := h.filterAliases(ctx, req)
	if err != nil {
		return nil, err
	}
	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for i, act := range req.Actions {
		var a txbuilder.Action
		switch act.Action.(type) {
		case *pb.Action_ControlAccount_:
			a, err = h.Accounts.DecodeControlAction(act.GetControlAccount())
		case *pb.Action_SpendAccount_:
			a, err = h.Accounts.DecodeSpendAction(act.GetSpendAccount())
		case *pb.Action_ControlProgram_:
			a, err = txbuilder.DecodeControlProgramAction(act.GetControlProgram())
		case *pb.Action_Issue_:
			a, err = h.Assets.DecodeIssueAction(act.GetIssue())
		case *pb.Action_SpendAccountUnspentOutput_:
			a, err = h.Accounts.DecodeSpendUTXOAction(act.GetSpendAccountUnspentOutput())
		case *pb.Action_SetTxReferenceData_:
			a, err = txbuilder.DecodeSetTxRefDataAction(act.GetSetTxReferenceData())
		}
		if err != nil {
			return nil, errors.WithDetailf(err, "on action %d", i)
		}
		actions = append(actions, a)
	}

	ttl := defaultTxTTL
	if req.Ttl != "" {
		ttl, err = time.ParseDuration(req.Ttl)
		if err != nil {
			return nil, errors.WithDetailf(httpjson.ErrBadRequest, "bad timeout %s", req.Ttl)
		}
	}
	maxTime := time.Now().Add(ttl)

	var txdata *bc.TxData

	if len(req.Transaction) > 0 {
		txdata, err = bc.NewTxDataFromBytes(req.Transaction)
		if err != nil {
			return nil, errors.WithDetailf(httpjson.ErrBadRequest, "bad tx data")
		}
	}

	tpl, err := txbuilder.Build(ctx, txdata, actions, maxTime)
	if errors.Root(err) == txbuilder.ErrAction {
		err = errors.WithData(err, "actions", errInfoBodyList(errors.Data(err)["actions"].([]error)))
	}
	if err != nil {
		return nil, err
	}

	return &pb.TxsResponse_Response{Template: tpl}, nil
}

func (h *Handler) BuildTxs(ctx context.Context, in *pb.BuildTxsRequest) (*pb.TxsResponse, error) {
	// If we're not the leader, we don't have access to the current
	// reservations. Forward the build call to the leader process.
	// TODO(jackson): Distribute reservations across cored processes.
	if !leader.IsLeading() {
		conn, err := leaderConn(ctx, h.DB, h.Addr)
		if err != nil {
			return nil, err
		}
		defer conn.Conn.Close()
		return pb.NewAppClient(conn.Conn).BuildTxs(ctx, in)
	}

	responses := make([]*pb.TxsResponse_Response, len(in.Requests))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(func(err error) {
				responses[i] = &pb.TxsResponse_Response{Error: protobufErr(err)}
			})

			tmpl, err := h.buildSingle(subctx, in.Requests[i])
			if err != nil {
				responses[i] = &pb.TxsResponse_Response{Error: protobufErr(err)}
			} else {
				responses[i] = tmpl
			}
		}(i)
	}

	wg.Wait()
	return &pb.TxsResponse{Responses: responses}, nil
}

func (h *Handler) submitSingle(ctx context.Context, tpl *pb.TxTemplate, waitUntil string) (*pb.SubmitTxsResponse_Response, error) {
	if len(tpl.RawTransaction) == 0 {
		return nil, errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	tx, err := bc.NewTxDataFromBytes(tpl.RawTransaction)
	if err != nil {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "bad transaction template")
	}

	err = h.finalizeTxWait(ctx, tpl, tx, waitUntil)
	if err != nil {
		return nil, errors.Wrapf(err, "tx %s", tx.Hash())
	}

	return &pb.SubmitTxsResponse_Response{Id: tx.Hash().String()}, nil
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
func (h *Handler) finalizeTxWait(ctx context.Context, txTemplate *pb.TxTemplate, txdata *bc.TxData, waitUntil string) error {
	// Use the current generator height as the lower bound of the block height
	// that the transaction may appear in.
	generatorHeight, _ := fetch.GeneratorHeight()
	localHeight := h.Chain.Height()
	if localHeight > generatorHeight {
		generatorHeight = localHeight
	}

	// Remember this height in case we retry this submit call.
	tx := bc.NewTx(*txdata)
	height, err := recordSubmittedTx(ctx, h.DB, tx.Hash, generatorHeight)
	if err != nil {
		return errors.Wrap(err, "saving tx submitted height")
	}

	err = txbuilder.FinalizeTx(ctx, h.Chain, h.Submitter, tx)
	if err != nil {
		return err
	}
	if waitUntil == "none" {
		return nil
	}

	height, err = h.waitForTxInBlock(ctx, tx, height)
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

func (h *Handler) waitForTxInBlock(ctx context.Context, tx *bc.Tx, height uint64) (uint64, error) {
	for {
		height++
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		case <-h.Chain.BlockWaiter(height):
			b, err := h.Chain.GetBlock(ctx, height)
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
			err = txbuilder.FinalizeTx(ctx, h.Chain, h.Submitter, tx)
			if err != nil {
				return 0, err
			}

			// TODO(jackson): Do simple rejection checks like checking if
			// the tx's blockchain prevouts still exist in the state tree.
		}
	}
}

func (h *Handler) SubmitTxs(ctx context.Context, in *pb.SubmitTxsRequest) (*pb.SubmitTxsResponse, error) {
	if !leader.IsLeading() {
		conn, err := leaderConn(ctx, h.DB, h.Addr)
		if err != nil {
			return nil, err
		}
		defer conn.Conn.Close()
		return pb.NewAppClient(conn.Conn).SubmitTxs(ctx, in)
	}

	// Setup a timeout for the provided wait duration.
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	responses := make([]*pb.SubmitTxsResponse_Response, len(in.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(func(err error) {
				responses[i] = &pb.SubmitTxsResponse_Response{Error: protobufErr(err)}
			})

			tx, err := h.submitSingle(subctx, in.Transactions[i], in.WaitUntil)
			if err != nil {
				responses[i] = &pb.SubmitTxsResponse_Response{Error: protobufErr(err)}
			} else {
				responses[i] = tx
			}
		}(i)
	}

	wg.Wait()
	return &pb.SubmitTxsResponse{Responses: responses}, nil
}
