package txscripttest

import (
	"fmt"

	"chain/cos/bc"
	"chain/cos/txscript"
)

var testTxHash = bc.Hash{}

// NewTestTx constructs a fresh TestTx.
func NewTestTx() *TestTx {
	return &TestTx{}
}

// TestTx is used to build a bc.TxData  in order to test the execution of a
// pkscript.
type TestTx struct {
	data      bc.TxData
	utxoIndex uint32
}

// AddInput adds a new input to the transaction.
func (tx *TestTx) AddInput(assetAmount bc.AssetAmount, pkscript []byte, inputWitness [][]byte) *TestTx {
	tx.utxoIndex++

	// Add the tx input to the current transaction.
	tx.data.Inputs = append(tx.data.Inputs, bc.NewSpendInput(testTxHash, tx.utxoIndex, inputWitness, assetAmount.AssetID, assetAmount.Amount, pkscript, nil))
	return tx
}

// AddOutput adds a new output to the transaction.
func (tx *TestTx) AddOutput(assetAmount bc.AssetAmount, pkscript []byte) *TestTx {
	tx.data.Outputs = append(tx.data.Outputs, bc.NewTxOutput(assetAmount.AssetID, assetAmount.Amount, pkscript, nil))
	return tx
}

// Execute constructs a new txscript.Engine and executes the pkscript for
// the input at the provided index.
func (tx *TestTx) Execute(inputIndex int) error {
	if inputIndex >= len(tx.data.Inputs) {
		return fmt.Errorf("input index %d; tx only has %d inputs", inputIndex, len(tx.data.Inputs))
	}

	input := tx.data.Inputs[inputIndex]
	vm, err := txscript.NewEngine(input.ControlProgram(), &tx.data, inputIndex, 0)
	if err != nil {
		return err
	}
	return vm.Execute()
}
