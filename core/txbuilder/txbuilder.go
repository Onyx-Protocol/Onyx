package txbuilder

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/ed25519/hd25519"
	"chain/errors"
)

var (
	// ErrBadBuildRequest is returned from Build when the arguments are
	// invalid.
	ErrBadBuildRequest = errors.New("bad build request")

	ErrNoSigScript = errors.New("data only for redeeming, not scripts")
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and destinations unsatisfied.
// Build partners then satisfy and consume inputs and destinations.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, prev *Template, sources []*Source, dests []*Destination, metadata []byte, ttl time.Duration) (*Template, error) {
	if ttl < time.Minute {
		ttl = time.Minute
	}
	tpl, err := build(ctx, sources, dests, metadata, ttl)
	if err != nil {
		return nil, err
	}
	if prev != nil {
		tpl, err = combine(prev, tpl)
		if err != nil {
			return nil, err
		}
	}

	ComputeSigHashes(ctx, tpl)
	return tpl, nil
}

func build(ctx context.Context, sources []*Source, dests []*Destination, metadata []byte, ttl time.Duration) (*Template, error) {
	tx := &bc.TxData{
		Version:  bc.CurrentTransactionVersion,
		Metadata: metadata,
	}

	var inputs []*Input

	for _, source := range sources {
		reserveResult, err := source.Reserve(ctx, ttl)
		if err != nil {
			return nil, errors.Wrap(err, "reserve")
		}
		for _, item := range reserveResult.Items {
			// Empty signature arrays should be serialized as empty arrays, not null.
			if item.TemplateInput.SigComponents == nil {
				item.TemplateInput.SigComponents = []*SigScriptComponent{}
			}

			tx.Inputs = append(tx.Inputs, item.TxInput)
			inputs = append(inputs, item.TemplateInput)
		}
		dests = append(dests, reserveResult.Change...)
	}

	for _, dest := range dests {
		tx.Outputs = append(tx.Outputs, bc.NewTxOutput(dest.AssetID, dest.Amount, dest.PKScript(), dest.Metadata))
	}

	appTx := &Template{
		Unsigned:   tx,
		BlockChain: "sandbox",
		Inputs:     inputs,
	}

	return appTx, nil
}

func combine(txs ...*Template) (*Template, error) {
	if len(txs) == 0 {
		return nil, errors.New("must pass at least one tx")
	}
	completeWire := &bc.TxData{Version: bc.CurrentTransactionVersion}
	complete := &Template{BlockChain: txs[0].BlockChain, Unsigned: completeWire}

	for _, tx := range txs {
		if tx.BlockChain != complete.BlockChain {
			return nil, errors.New("all txs must be the same BlockChain")
		}

		if len(tx.Unsigned.Metadata) != 0 {
			if len(complete.Unsigned.Metadata) != 0 &&
				!bytes.Equal(tx.Unsigned.Metadata, complete.Unsigned.Metadata) {
				return nil, errors.WithDetail(ErrBadBuildRequest, "transaction metadata does not match previous template's metadata")
			}

			complete.Unsigned.Metadata = tx.Unsigned.Metadata
		}

		complete.Inputs = append(complete.Inputs, tx.Inputs...)
		completeWire.Inputs = append(completeWire.Inputs, tx.Unsigned.Inputs...)
		completeWire.Outputs = append(completeWire.Outputs, tx.Unsigned.Outputs...)
	}

	return complete, nil
}

// ComputeSigHashes populates signature data for every input and sigscript
// component.
func ComputeSigHashes(ctx context.Context, tpl *Template) {
	sigHasher := bc.NewSigHasher(tpl.Unsigned)
	for i, in := range tpl.Inputs {
		h := sigHasher.Hash(i, bc.SigHashAll)
		for _, c := range in.SigComponents {
			c.SignatureData = h
		}
	}
}

// AssembleSignatures takes a filled in Template
// and adds the signatures to the template's unsigned transaction,
// creating a fully-signed transaction.
func AssembleSignatures(txTemplate *Template) (*bc.Tx, error) {
	msg := txTemplate.Unsigned
	for i, input := range txTemplate.Inputs {
		if msg.Inputs[i] == nil {
			return nil, fmt.Errorf("unsigned tx missing input %d", i)
		}

		components := input.SigComponents

		witness := make([][]byte, 0, len(components))

		for _, c := range components {
			switch c.Type {
			case "script":
				return nil, ErrNoSigScript
			case "data":
				witness = append(witness, c.Data)
			case "signature":
				added := 0
				for _, sig := range c.Signatures {
					if len(sig.Bytes) == 0 {
						continue
					}
					witness = append(witness, sig.Bytes)
					added++
					if added == c.Required {
						break
					}
				}
			default:
				return nil, fmt.Errorf("unknown sigscript component `%s`", c.Type)
			}
		}
		msg.Inputs[i].InputWitness = witness
	}

	return bc.NewTx(*msg), nil
}

// InputSigs takes a set of keys
// and creates a matching set of Input Signatures
// for a Template
func InputSigs(keys []*hd25519.XPub, path []uint32) (sigs []*Signature) {
	sigs = []*Signature{}
	for _, k := range keys {
		sigs = append(sigs, &Signature{
			XPub:           k.String(),
			DerivationPath: path,
		})
	}
	return sigs
}

func getSigsRequired(script []byte) (sigsReqd int, err error) {
	sigsReqd = 1
	if txscript.IsMultiSig(script) {
		_, sigsReqd, err = txscript.CalcMultiSigStats(script)
		if err != nil {
			return 0, err
		}
	}
	return sigsReqd, nil
}

func Sign(ctx context.Context, tpl *Template, signFn func(context.Context, *SigScriptComponent, *Signature) ([]byte, error)) error {
	ComputeSigHashes(ctx, tpl)
	// TODO(kr): come up with some scheme to verify that the
	// covered output scripts are what the client really wants.
	for i, input := range tpl.Inputs {
		if len(input.SigComponents) > 0 {
			for c, component := range input.SigComponents {
				if component.Type != "signature" {
					continue
				}
				for s, sig := range component.Signatures {
					sigBytes, err := signFn(ctx, component, sig)
					if err != nil {
						return errors.Wrapf(err, "computing signature for input %d, sigscript component %d, sig %d", i, c, s)
					}
					sig.Bytes = append(sigBytes, byte(bc.SigHashAll))
				}
			}
		}
	}
	return nil
}
