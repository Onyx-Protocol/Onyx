package bcvm

import (
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/txvm"
)

type TimeConstraint struct {
	Type string
	Time int64
}

type Nonce struct {
	ID               bc.Hash
	MinTime, MaxTime int64
	Program          []byte
}

type Value struct {
	Amount  int64
	AssetID bc.Hash
}

type Output struct {
	ID      bc.Hash
	History bc.Hash
	Values  []Value
	Program []byte
}

type Tx struct {
	ID              bc.Hash
	TimeConstraints []TimeConstraint
	Anchors         []bc.Hash
	Nonces          []Nonce
	Inputs          []Output
	Outputs         []Output
	Annotations     [][]byte
	Program         []byte
}

func NewTx(p []byte) (*Tx, error) {
	tx := new(Tx)
	id, ok := txvm.Validate(p, txvm.TraceOp(tx.trace))
	if !ok {
		return nil, errors.New("invalid transaction")
	}
	tx.ID = bc.NewHash(id)
	return tx, nil
}

func (tx *Tx) trace(op byte, _ []byte, vm txvm.VM) {
	if op != txvm.Summarize {
		return
	}

	tcs := vm.Stack(txvm.StackTimeConstraint)
	for i := 0; i < tcs.Len(); i++ {
		tuple := tcs.Element(i).(txvm.Tuple)
		switch string(tuple.Field(0).(txvm.Bytes)) {
		case txvm.MinTimeTuple:
			tx.TimeConstraints = append(tx.TimeConstraints, TimeConstraint{
				Type: "min",
				Time: int64(tuple.Field(1).(txvm.Int64)),
			})
		case txvm.MaxTimeTuple:
			tx.TimeConstraints = append(tx.TimeConstraints, TimeConstraint{
				Type: "max",
				Time: int64(tuple.Field(1).(txvm.Int64)),
			})
		}
	}

	nonces := vm.Stack(txvm.StackNonce)
	for i := 0; i < nonces.Len(); i++ {
		tuple := nonces.Element(i).(txvm.Tuple)
		var id [32]byte
		copy(nonces.ID(i), id[:])
		tx.Nonces = append(tx.Nonces, Nonce{
			ID:      bc.NewHash(id),
			MinTime: int64(tuple.Field(2).(txvm.Int64)),
			MaxTime: int64(tuple.Field(3).(txvm.Int64)),
			Program: tuple.Field(1).(txvm.Bytes),
		})
	}

	stackOutputs := func(stack txvm.Stack) []Output {
		var outputs []Output

		for i := 0; i < stack.Len(); i++ {
			tuple := stack.Element(i).(txvm.Tuple)
			var (
				id      [32]byte
				history [32]byte
			)
			copy(stack.ID(i), id[:])
			copy(tuple.Field(1).(txvm.Bytes), history[:])

			var values []Value
			tupleVals := tuple.Field(2).(txvm.Tuple)
			for j := 0; j < tupleVals.Len(); j++ {
				valueTuple := tupleVals.Field(j).(txvm.Tuple)
				var assetID [32]byte
				copy(valueTuple.Field(1).(txvm.Bytes), assetID[:])
				values = append(values, Value{
					Amount:  int64(valueTuple.Field(0).(txvm.Int64)),
					AssetID: bc.NewHash(assetID),
				})
			}

			outputs = append(outputs, Output{
				ID:      bc.NewHash(id),
				History: bc.NewHash(history),
				Values:  []Value{},
				Program: tuple.Field(3).(txvm.Bytes),
			})
		}

		return outputs
	}

	tx.Inputs = stackOutputs(vm.Stack(txvm.StackInput))
	tx.Outputs = stackOutputs(vm.Stack(txvm.StackOutput))

	annotations := vm.Stack(txvm.StackAnnotation)
	for i := 0; i < annotations.Len(); i++ {
		tuple := annotations.Element(i).(txvm.Tuple)
		tx.Annotations = append(tx.Annotations, tuple.Field(1).(txvm.Bytes))
	}
}
