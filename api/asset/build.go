package asset

import (
	"chain/api/utxodb"
	"time"

	"golang.org/x/net/context"
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and outputs unsatisfied.
// Build partners then satisfy and consume inputs and outputs.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, prev *Tx, inputs []utxodb.Input, outputs []*Output, ttl time.Duration) (*Tx, error) {
	if ttl < time.Minute {
		ttl = time.Minute
	}
	tpl, err := build(ctx, inputs, outputs, ttl)
	if err != nil {
		return nil, err
	}
	if prev != nil {
		return combine(prev, tpl)
	}
	return tpl, nil
}
