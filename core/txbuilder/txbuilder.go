package txbuilder

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
	"chain/errors"
)

// ErrBadBuildRequest is returned from Build when the
// arguments are invalid.
var ErrBadBuildRequest = errors.New("bad build request")

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
			if item.TemplateInput.Sigs == nil {
				item.TemplateInput.Sigs = []*Signature{}
			}
			if item.TemplateInput.SigComponents == nil {
				item.TemplateInput.SigComponents = []*SigScriptComponent{}
			}

			tx.Inputs = append(tx.Inputs, item.TxInput)
			inputs = append(inputs, item.TemplateInput)
		}
		dests = append(dests, reserveResult.Change...)
	}

	for _, dest := range dests {
		output := &bc.TxOutput{
			AssetAmount: bc.AssetAmount{AssetID: dest.AssetID, Amount: dest.Amount},
			Script:      dest.PKScript(),
			Metadata:    dest.Metadata,
		}
		tx.Outputs = append(tx.Outputs, output)
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
	hashCache := &bc.SigHashCache{}
	for i, in := range tpl.Inputs {
		aa := in.AssetAmount
		in.SignatureData = tpl.Unsigned.HashForSigCached(i, aa, bc.SigHashAll, hashCache)
		for _, c := range in.SigComponents {
			c.SignatureData = in.SignatureData
		}
	}
}

// AssembleSignatures takes a filled in Template
// and adds the signatures to the template's unsigned transaction,
// creating a fully-signed transaction.
func AssembleSignatures(txTemplate *Template) (*bc.Tx, error) {
	msg := txTemplate.Unsigned
	for i, input := range txTemplate.Inputs {

		components := input.SigComponents

		// For backwards compatability, convert old input.Sigs to a signature
		// sigscript component.
		// TODO(jackson): Remove once all the SDKs are using the new format.
		if len(input.Sigs) > 0 || len(input.SigComponents) == 0 {
			sigsReqd, err := getSigsRequired(input.SigScriptSuffix)
			if err != nil {
				return nil, err
			}

			// Replace the existing components. Only SDKs that don't understand
			// signature components will populate input.Sigs.
			components = []*SigScriptComponent{
				{
					Type:          "signature",
					Required:      sigsReqd,
					SignatureData: input.SignatureData,
					Signatures:    input.Sigs,
				},
				{
					Type:   "script",
					Script: input.SigScriptSuffix,
				},
			}
		}

		sb := txscript.NewScriptBuilder()
		for _, c := range components {
			switch c.Type {
			case "script":
				sb.ConcatRawScript(c.Script)
			case "data":
				sb.AddData(c.Data)
			case "signature":
				if len(c.Signatures) == 0 {
					break
				}

				sb.AddOp(txscript.OP_FALSE)
				added := 0
				for _, sig := range c.Signatures {
					if len(sig.DER) == 0 {
						continue
					}
					sb.AddData(sig.DER)
					added++
					if added == c.Required {
						break
					}
				}
			default:
				return nil, fmt.Errorf("unknown sigscript component `%s`", c.Type)
			}
		}
		script, err := sb.Script()
		if err != nil {
			return nil, errors.Wrap(err)
		}
		msg.Inputs[i].SignatureScript = script
	}
	return bc.NewTx(*msg), nil
}

// InputSigs takes a set of keys
// and creates a matching set of Input Signatures
// for a Template
func InputSigs(keys []*hdkey.Key) (sigs []*Signature) {
	sigs = []*Signature{}
	for _, k := range keys {
		sigs = append(sigs, &Signature{
			XPub:           k.Root.String(),
			DerivationPath: k.Path,
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
