package asset

import (
	"encoding/json"

	"golang.org/x/net/context"

	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type ScriptReceiver struct {
	script []byte
}

func (receiver *ScriptReceiver) PKScript() []byte { return receiver.script }
func (receiver *ScriptReceiver) AccumulateUTXO(ctx context.Context, outpoint *bc.Outpoint, txOutput *bc.TxOutput, utxoInserters []txbuilder.UTXOInserter) ([]txbuilder.UTXOInserter, error) {
	// Find or create an item in utxoInserters that is a
	// ScriptUTXOInserter
	var scriptUTXOInserter *ScriptUTXOInserter
	for _, inserter := range utxoInserters {
		var ok bool
		if scriptUTXOInserter, ok = inserter.(*ScriptUTXOInserter); ok {
			break
		}
	}
	if scriptUTXOInserter == nil {
		scriptUTXOInserter = &ScriptUTXOInserter{}
		utxoInserters = append(utxoInserters, scriptUTXOInserter)
	}
	scriptUTXOInserter.Add(outpoint, txOutput, receiver)
	return utxoInserters, nil
}

func (receiver *ScriptReceiver) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"script": chainjson.HexBytes(receiver.script),
		"type":   "script",
	})
}

func NewScriptReceiver(script []byte) *ScriptReceiver {
	return &ScriptReceiver{
		script: script,
	}
}

func NewScriptDestination(ctx context.Context, assetAmount *bc.AssetAmount, script []byte, metadata []byte) (*txbuilder.Destination, error) {
	scriptReceiver := NewScriptReceiver(script)
	dest := &txbuilder.Destination{
		AssetAmount: *assetAmount,
		Metadata:    metadata,
		Receiver:    scriptReceiver,
	}
	return dest, nil
}

type (
	ScriptUTXOPair struct {
		outpoint *bc.Outpoint
		txOutput *bc.TxOutput
	}
	ScriptUTXOInserter struct {
		pairs []*ScriptUTXOPair
	}
)

func (inserter *ScriptUTXOInserter) Add(outpoint *bc.Outpoint, txOutput *bc.TxOutput, receiver *ScriptReceiver) {
	pair := &ScriptUTXOPair{
		outpoint: outpoint,
		txOutput: txOutput,
	}
	inserter.pairs = append(inserter.pairs, pair)
}

func (inserter *ScriptUTXOInserter) InsertUTXOs(ctx context.Context) ([]*txdb.Output, error) {
	txdbOutputs := make([]*txdb.Output, 0, len(inserter.pairs))
	var (
		scripts             [][]byte
		txdbOutputsByScript = make(map[string]*txdb.Output)
	)
	for _, pair := range inserter.pairs {
		txOutput := pair.txOutput
		script := txOutput.Script
		txdbOutput := &txdb.Output{
			Output: state.Output{
				TxOutput: *txOutput,
				Outpoint: *pair.outpoint,
			},
		}
		txdbOutputs = append(txdbOutputs, txdbOutput)
		scripts = append(scripts, script)
		txdbOutputsByScript[string(script)] = txdbOutput
	}

	// Load account ID, manager node ID, and addr index from the
	// addresses table for outputs that need it.  Not all are guaranteed
	// to be in the database; some outputs will be owned by third
	// parties.  This function loads what it can.
	const q = `
		SELECT pk_script, account_id, manager_node_id, key_index(key_index)
		FROM addresses
		WHERE pk_script IN (SELECT unnest($1::bytea[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Byteas(scripts))
	if err != nil {
		return nil, errors.Wrap(err, "select")
	}
	defer rows.Close()
	for rows.Next() {
		var (
			script        []byte
			managerNodeID string
			accountID     string
			addrIndex     []uint32
		)
		err = rows.Scan(
			&script,
			&accountID,
			&managerNodeID,
			(*pg.Uint32s)(&addrIndex),
		)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		txdbOutput := txdbOutputsByScript[string(script)]
		txdbOutput.AccountID = accountID
		txdbOutput.ManagerNodeID = managerNodeID
		copy(txdbOutput.AddrIndex[:], addrIndex)
	}
	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "rows")
	}

	return txdbOutputs, txdb.InsertPoolOutputs(ctx, txdbOutputs)
}
