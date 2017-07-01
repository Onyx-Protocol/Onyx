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
	AssetID bc.AssetID
}

type Output struct {
	ID      bc.Hash
	Anchor  bc.Hash
	Values  []Value
	Program []byte
}

type Tx struct {
	ID              bc.Hash
	TimeConstraints []TimeConstraint
	IssueAnchors    []bc.Hash
	Nonces          []Nonce
	Inputs          []Output
	Outputs         []Output
	Annotations     [][]byte
	Program         []byte
}

func NewTx(p []byte) (*Tx, error) {
	tx := new(Tx)
	var err error
	id, ok := txvm.Validate(p, txvm.TraceOp(tx.trace), txvm.TraceError(func(e error) {
		err = e
	}))
	tx.ID = bc.NewHash(id)
	if err != nil {
		return tx, errors.Wrap(err, "invalid transaction")
	}
	if !ok {
		return tx, errors.New("invalid transaction")
	}
	return tx, nil
}

func (tx *Tx) trace(op byte, _ []byte, vm txvm.VM) {
	switch op {
	case txvm.Summarize:
		tx.traceSummarize(vm)
	case txvm.Issue:
		tx.traceIssue(vm)
	}
}

func (tx *Tx) traceSummarize(vm txvm.VM) {
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
		copy(id[:], nonces.ID(i))
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
				id     [32]byte
				anchor [32]byte
			)
			copy(id[:], stack.ID(i))
			copy(anchor[:], tuple.Field(3).(txvm.Bytes))

			var values []Value
			tupleVals := tuple.Field(1).(txvm.Tuple)
			for j := 0; j < tupleVals.Len(); j++ {
				valueTuple := tupleVals.Field(j).(txvm.Tuple)
				var assetID [32]byte
				copy(assetID[:], valueTuple.Field(1).(txvm.Bytes))
				values = append(values, Value{
					Amount:  int64(valueTuple.Field(0).(txvm.Int64)),
					AssetID: bc.NewAssetID(assetID),
				})
			}

			outputs = append(outputs, Output{
				ID:      bc.NewHash(id),
				Anchor:  bc.NewHash(anchor),
				Values:  []Value{},
				Program: tuple.Field(2).(txvm.Bytes),
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

func (tx *Tx) traceIssue(vm txvm.VM) {
	stack := vm.Stack(txvm.StackAnchor)
	var id [32]byte
	copy(id[:], stack.ID(stack.Len()-1))
	tx.IssueAnchors = append(tx.IssueAnchors, bc.NewHash(id))
}
