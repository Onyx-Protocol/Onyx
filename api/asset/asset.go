// Package asset provides business logic for manipulating assets.
package asset

import (
	"database/sql"
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/crypto/hash256"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	chaintxscript "chain/fedchain-sandbox/txscript"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/metrics"
)

// ErrBadAddr is returned by Issue.
var ErrBadAddr = errors.New("bad address")

// Issue creates a transaction that
// issues new units of an asset
// distributed to the outputs provided.
func Issue(ctx context.Context, assetID string, dests []*Destination) (*TxTemplate, error) {
	defer metrics.RecordElapsed(time.Now())

	hash, err := bc.ParseHash(assetID)
	assetHash := bc.AssetID(hash)

	asset, err := appdb.AssetByID(ctx, assetHash)
	if err != nil {
		return nil, errors.WithDetailf(err, "get asset with ID %q", assetID)
	}

	tx := &bc.TxData{Version: bc.CurrentTransactionVersion}
	in := &bc.TxInput{Previous: bc.Outpoint{
		Index: bc.InvalidOutputIndex,
		Hash:  bc.Hash{}, // TODO(kr): figure out anti-replay for issuance
	}}

	if len(asset.Definition) != 0 {
		defHash, err := txdb.DefinitionHashByAssetID(ctx, assetID)
		if err != nil && errors.Root(err) != sql.ErrNoRows {
			return nil, errors.WithDetailf(err, "get asset definition pointer for %s", assetID)
		}

		newDefHash := bc.Hash(hash256.Sum(asset.Definition)).String()
		if defHash != newDefHash {
			in.AssetDefinition = asset.Definition
		}
	}

	tx.Inputs = append(tx.Inputs, in)

	outRecvs, err := addAssetIssuanceOutputs(ctx, tx, asset, dests)
	if err != nil {
		return nil, errors.Wrap(err, "add issuance outputs")
	}

	input, err := issuanceInput(asset, tx)
	if err != nil {
		return nil, errors.Wrap(err, "creating issuance Input")
	}

	appTx := &TxTemplate{
		Unsigned:   tx,
		BlockChain: "sandbox", // TODO(tess): make this BlockChain: blockchain.FromContext(ctx)
		Inputs:     []*Input{input},
		OutRecvs:   outRecvs,
	}
	return appTx, nil
}

// Destination is a user input struct that describes
// the destination of a transaction's inputs.
type Destination struct {
	AssetID    bc.AssetID `json:"asset_id"`
	Amount     uint64
	Metadata   chainjson.HexBytes
	Type       string // determines pkScripter type
	pkScripter pkScripter
}

// AccountID returns the account ID for this destination, if any.
// It can be used for authorization.
func (d *Destination) AccountID() string {
	if a, ok := d.pkScripter.(*acctPKScripter); ok {
		return a.AccountID
	}
	return ""
}

func (d *Destination) UnmarshalJSON(b []byte) error {
	type dest Destination // lose the method set; avoid infinite recursion
	err := json.Unmarshal(b, (*dest)(d))
	if err != nil {
		return err
	}
	switch d.Type {
	case "account", "": // default type
		d.pkScripter = new(acctPKScripter)
	case "address":
		// TODO(kr): move to a "script" type
		// containing the literal pk script.
		d.pkScripter = new(addrPKScripter)
	default:
		return errors.WithDetailf(ErrBadOutDest, "unknown type %q", d.Type)
	}
	return json.Unmarshal(b, d.pkScripter)
}

// pkScripter computes the PK script (aka output script)
// for sending to an arbitrary destination.
// The returned recv can be nil.
type pkScripter interface {
	pkScript(context.Context) (script []byte, recv *utxodb.Receiver, err error)
}

type acctPKScripter struct {
	AccountID string `json:"account_id"`
	isChange  bool
}

// pkScript returns the script for sending to
// the destination account id in s.
func (s *acctPKScripter) pkScript(ctx context.Context) ([]byte, *utxodb.Receiver, error) {
	addr := &appdb.Address{
		AccountID: s.AccountID,
		IsChange:  s.isChange,
	}
	err := CreateAddress(ctx, addr, false)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "output create address error account=%v", s.AccountID)
	}
	return addr.PKScript, newOutputReceiver(addr, s.isChange), nil
}

type addrPKScripter struct {
	Address  string `json:"address"`
	isChange bool
}

// pkScript returns the script for sending to
// the destination address in s.
// The returned extra data is nil.
func (s *addrPKScripter) pkScript(context.Context) ([]byte, *utxodb.Receiver, error) {
	script, err := chaintxscript.AddrPkScript(s.Address)
	if err != nil {
		return nil, nil, errors.Wrapf(ErrBadAddr, "output pkscript error addr=%v", s.Address)
	}
	return script, nil, nil
}

func addAssetIssuanceOutputs(ctx context.Context, tx *bc.TxData, asset *appdb.Asset, dests []*Destination) ([]*utxodb.Receiver, error) {
	var outAddrs []*utxodb.Receiver
	for i, dest := range dests {
		pkScript, receiver, err := dest.pkScripter.pkScript(ctx)
		if err != nil {
			return nil, errors.WithDetailf(err, "output %d", i)
		}
		tx.Outputs = append(tx.Outputs, &bc.TxOutput{
			AssetID:  asset.Hash,
			Value:    dest.Amount,
			Script:   pkScript,
			Metadata: dest.Metadata,
		})
		outAddrs = append(outAddrs, receiver)
	}
	return outAddrs, nil
}

func newOutputReceiver(addr *appdb.Address, isChange bool) *utxodb.Receiver {
	return &utxodb.Receiver{
		ManagerNodeID: addr.ManagerNodeID,
		AccountID:     addr.AccountID,
		AddrIndex:     addr.Index,
		IsChange:      isChange,
	}
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *appdb.Asset, tx *bc.TxData) (*Input, error) {
	hash, err := txscript.CalcSignatureHash(tx, 0, a.RedeemScript, txscript.SigHashAll)
	if err != nil {
		return nil, errors.Wrap(err, "calculating signature hash")
	}

	return &Input{
		IssuerNodeID:  a.IssuerNodeID,
		RedeemScript:  a.RedeemScript,
		SignatureData: hash,
		Sigs:          inputSigs(hdkey.Derive(a.Keys, appdb.IssuancePath(a))),
	}, nil
}

func inputSigs(keys []*hdkey.Key) (sigs []*Signature) {
	for _, k := range keys {
		sigs = append(sigs, &Signature{
			XPub:           k.Root.String(),
			DerivationPath: k.Path,
		})
	}
	return sigs
}
