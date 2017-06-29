package protocol

import (
	"chain/errors"
	"chain/protocol/validation"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx []byte) error {
	err := validation.ValidateTx(tx)
	return errors.Sub(ErrBadTx, err)
}
