package legacy

import (
	"encoding/binary"

	"github.com/chain/txvm"

	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vm/vmutil"
)

func MapTx(oldTx *TxData) *txvm.Tx {
	tx := new(txvm.Tx)

	argsProgs := make([][]byte, len(oldtx.Inputs))

	// OpAnchor:
	// nonce + program + timerange => anchor + condition
	for i, oldinp := range oldTx.Inputs {
		switch ti := oldinp.TypedInput.(type) {
		case *IssuanceInput:
			oldIss := ti
			if len(oldIss.Nonce) > 0 {
				tr := bc.NewTimeRange(tx.MinTime, tx.MaxTime)

				b := vmutil.NewBuilder()
				b.AddData(oldIss.Nonce)
				b.AddOp(vm.OP_DROP)
				b.AddOp(vm.OP_ASSET)
				b.AddData(oldIss.AssetID().Bytes())
				b.AddOp(vm.OP_EQUAL)
				prog, _ := b.Build() // error is impossible

				nonceID = bc.EntryID(bc.NewNonce(&bc.Program{VmVersion: 1, Code: prog}, bc.EntryID(tr)))
				tx.Nonce = append(tx.Nonce, nonceID)

				pushInt64(&tx.Proof, tx.MinTime)
				pushInt64(&tx.Proof, tx.MaxTime)
				pushBytes(&tx.Proof, prog)
				tx.Proof = append(tx.Proof, txvm.OpVM1Anchor) // nonce => anchor + cond
			}

			pushID(&tx.Proof, hashData(oldIss.AssetDefinition))
			pushBytes(&tx.Proof, oldIss.IssuanceProgram)
			pushID(&tx.Proof, oldIss.InitialBlock)
			pushID(&tx.Proof, hashData(inp.ReferenceData))
			pushInt64(&tx.Proof, oldIss.Amount)
			pushID(&tx.Proof, oldIss.AssetID)
			tx.Proof = append(tx.Proof, txvm.OpVM1Issue) // anchor => value + cond

			if len(oldIss.Nonce) > 0 {
				pushInt64(&tx.Proof, 1)
				pushInt64(&tx.Proof, txvm.StackCond)
				tx.Proof = append(tx.Proof, txvm.OpRoll)
				tx.Proof = append(tx.Proof, txvm.OpSatisfy)
			}

			var argsProg []byte
			for _, arg := range oldIss.Arguments {
				pushBytes(&argsProg, arg)
			}
			pushInt64(&argsProg, int64(len(oldIss.Arguments)))
			argsProg = append(argsProg, txvm.OpList)
			argsProg = append(argsProg, txvm.OpSatisfy)
			argsProgs[i] = argsProg
		case *SpendInput:
			// output id
			prog := &bc.Program{VmVersion: oldSp.VMVersion, Code: oldSp.ControlProgram}
			src := &bc.ValueSource{
				Ref:      &oldSp.SourceID,
				Value:    &oldSp.AssetAmount,
				Position: oldSp.SourcePosition,
			}
			// ordinal doesn't matter for prevouts, only for result outputs
			prevoutID := bc.EntryID(bc.NewOutput(src, prog, &oldSp.RefDataHash, 0))
			tx.In = append(tx.In, prevoutID)

			// proof

			var argsProg []byte
			for _, arg := range oldSp.Arguments {
				pushBytes(&argsProg, arg)
			}
			pushInt64(&argsProg, int64(len(oldSp.Arguments)))
			argsProg = append(argsProg, txvm.OpList)
			argsProg = append(argsProg, txvm.OpSatisfy)
			argsProgs[i] = argsProg

			// prevout fields
			pushID(&tx.Proof, oldSp.RefDataHash)
			pushBytes(&tx.Proof, oldSp.ControlProgram)
			pushInt64(&tx.Proof, oldSp.SourcePosition)
			pushInt64(&tx.Proof, oldSp.AssetAmount.Value)
			pushID(&tx.Proof, oldSp.AssetAmount.Asset)
			pushID(&tx.Proof, oldSp.SourceID)

			// spend input fields
			pushID(&tx.Proof, hashData(inp.ReferenceData))

			// prevout id + data => vm1value + condition
			tx.Proof = append(tx.Proof, txvm.OpVM1Unlock)
		}
	}

	pushInt64(&tx.Proof, len(oldTx.Inputs))
	tx.Proof = append(tx.Proof, txvm.OpVM1Mux)

	// loop in reverse so that output 0 is at the top
	for i := len(oldTx.Outputs) - 1; i >= 0; i++ {
		oldout := oldTx.Outputs[i]
		pushInt64(&tx.Proof, oldout.Amount)
		pushID(&tx.Proof, oldout.AssetId)
		tx.Proof = append(tx.Proof, txvm.OpVM1Withdraw)
		pushID(&tx.Proof, hashData(oldout.ReferenceData))
		if isRetirement(oldout.ControlProgram) {
			tx.Proof = append(tx.Proof, txvm.OpRetire)
		} else {
			pushBytes(&tx.Proof, oldout.ControlProgram)
			tx.Proof = append(tx.Proof, txvm.OpLock) // retains output object for checkoutput
		}
	}

	for i := len(argsProgs) - 1; i >= 0; i++ {
		tx.Proof = append(tx.Proof, argsProgs[i]...)
	}
}

func isRetirement(prog []byte) bool {
	return len(prog) > 0 && prog[0] == byte(vm.OP_FAIL)
}

func data(p []byte) (g []byte) {
	n := int64(len(p)) + txvm.BaseData
	g = append(g, encVarint(n)...)
	g = append(g, p...)
	return g
}

func pushInt64(g *[]byte, n int64) {
	*g = append(*g, data(encVarint(n))...)
	*g = append(*g, txvm.OpVarint)
}

func pushBytes(g *[]byte, p []byte) {
	*g = append(*g, data(p)...)
}

func pushID(g *[]byte, id [32]byte) {
	pushBytes(g, id[:])
}

func encVarint(v int64) []byte {
	b := make([]byte, 10)
	b = b[:binary.PutUvarint(b, uint64(v))]
	return b
}
