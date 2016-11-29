// Package txbuilder builds a Chain Protocol transaction from
// a list of actions.
package txbuilder

import (
	"context"
	"time"

	"chain-stealth/crypto/ed25519/chainkd"
	"chain-stealth/encoding/json"
	"chain-stealth/errors"
	"chain-stealth/protocol/bc"
)

var (
	ErrBadRefData            = errors.New("transaction reference data does not match previous template's reference data")
	ErrBadTxInputIdx         = errors.New("unsigned tx missing input")
	ErrBadWitnessComponent   = errors.New("invalid witness component")
	ErrBadAmount             = errors.New("bad asset amount")
	ErrBlankCheck            = errors.New("unsafe transaction: leaves assets free to control")
	ErrBadConfidentialityKey = errors.New("confidentiality key invalid")
	ErrAction                = errors.New("errors occurred in one or more actions")
	ErrMissingFields         = errors.New("required field is missing")
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and destinations unsatisfied.
// Build partners then satisfy and consume inputs and destinations.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, tx *bc.TxData, actions []Action, ci []*ConfidentialityInstruction, maxTime time.Time) (*Template, error) {
	builder := TemplateBuilder{
		base:             tx,
		confInstructions: ci,
	}
	builder.RestrictMaxTimeMS(bc.Millis(maxTime))

	// Build all of the actions, updating the builder.
	var errs []error
	for i, action := range actions {
		err := action.Build(ctx, maxTime, &builder)
		if err != nil {
			err = errors.WithData(err, "index", i)
			errs = append(errs, err)
		}
	}

	// If there were any errors, rollback and return a composite error.
	if len(errs) > 0 {
		builder.rollback()
		return nil, errors.WithData(ErrAction, "actions", errs)
	}

	// Build the transaction template.
	tpl, err := builder.Build()
	if err != nil {
		builder.rollback()
		return nil, err
	}

	return tpl, nil
}

// KeyIDs produces KeyIDs from a list of xpubs and a derivation path
// (applied to all the xpubs).
func KeyIDs(xpubs []chainkd.XPub, path [][]byte) []KeyID {
	result := make([]KeyID, 0, len(xpubs))
	var hexPath []json.HexBytes
	for _, p := range path {
		hexPath = append(hexPath, p)
	}
	for _, xpub := range xpubs {
		result = append(result, KeyID{xpub.String(), hexPath})
	}
	return result
}

func Sign(ctx context.Context, tpl *Template, xpubs []string, signFn SignFunc) error {
	for i, sigInst := range tpl.SigningInstructions {
		for j, c := range sigInst.WitnessComponents {
			err := c.Sign(ctx, tpl, i, xpubs, signFn)
			if err != nil {
				return errors.WithDetailf(err, "adding signature(s) to witness component %d of input %d", j, i)
			}
		}
	}
	return materializeWitnesses(tpl)
}

// MissingFieldsError returns a wrapped error ErrMissingFields
// with a data item containing the given field names.
func MissingFieldsError(name ...string) error {
	return errors.WithData(ErrMissingFields, "missing_fields", name)
}
