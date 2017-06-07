package legacy

import (
	"chain/protocol/txvm"

	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vm/vmutil"
)

func MapVMTx(oldTx *TxData) *txvm.Tx {
	tx := &txvm.Tx{
		MinTime: oldTx.MinTime,
		MaxTime: oldTx.MaxTime,
	}

	argsProgs := make([][]byte, len(oldTx.Inputs))

	// OpAnchor:
	// nonce + program + timerange => anchor + condition
	proof := txvm.Builder{}
	for _, oldinp := range oldTx.Inputs {
		switch ti := oldinp.TypedInput.(type) {
		case *IssuanceInput:
			oldIss := ti
			if len(oldIss.Nonce) > 0 {
				tr := bc.NewTimeRange(oldTx.MinTime, oldTx.MaxTime)

				b := vmutil.NewBuilder()
				b.AddData(oldIss.Nonce)
				b.AddOp(vm.OP_DROP)
				b.AddOp(vm.OP_ASSET)
				b.AddData(oldIss.AssetID().Bytes())
				b.AddOp(vm.OP_EQUAL)
				prog, _ := b.Build() // error is impossible

				trID := bc.EntryID(tr)

				nonceID := bc.EntryID(bc.NewNonce(&bc.Program{VmVersion: 1, Code: prog}, &trID))
				tx.Nonce = append(tx.Nonce, txvm.ID(nonceID.Byte32()))

				proof.Int64(int64(oldTx.MinTime)).Int64(int64(oldTx.MaxTime)).Data(prog).Op(txvm.Anchor) // nonce => anchor + cond

				var argsProg []byte
				argsProg = append(argsProg, txvm.Satisfy)
				argsProgs = append(argsProgs, argsProg)
			}
			proof.
				Data(hashData(oldIss.AssetDefinition).Bytes()).
				Data(oldIss.IssuanceProgram).
				Data(oldIss.InitialBlock.Bytes()).
				Data(hashData(oldinp.ReferenceData).Bytes()).
				Int64(int64(oldIss.Amount)).
				Data(oldIss.AssetID().Bytes()).
				Op(txvm.VM1Issue) // anchor => value + cond

			var argsProg txvm.Builder
			for _, arg := range oldIss.Arguments {
				argsProg.Data(arg)
			}
			argsProg.Int64(int64(len(oldIss.Arguments))).Op(txvm.Tuple).Op(txvm.Satisfy)
			argsProgs = append(argsProgs, argsProg.Build())
		case *SpendInput:
			oldSp := ti
			// output id
			prog := &bc.Program{VmVersion: oldSp.VMVersion, Code: oldSp.ControlProgram}
			src := &bc.ValueSource{
				Ref:      &oldSp.SourceID,
				Value:    &oldSp.AssetAmount,
				Position: oldSp.SourcePosition,
			}
			// ordinal doesn't matter for prevouts, only for result outputs
			prevoutID := bc.EntryID(bc.NewOutput(src, prog, &oldSp.RefDataHash, 0))
			tx.In = append(tx.In, prevoutID.Byte32())

			// proof

			var argsProg txvm.Builder
			for _, arg := range oldSp.Arguments {
				argsProg.Data(arg)
			}
			argsProg.Int64(int64(len(oldSp.Arguments))).Op(txvm.Tuple).Op(txvm.Satisfy)
			argsProgs = append(argsProgs, argsProg.Build())

			// prevout fields
			proof.
				Data(oldSp.RefDataHash.Bytes()).
				Data(oldSp.ControlProgram).
				Int64(int64(oldSp.SourcePosition)).
				Int64(int64(oldSp.Amount)).
				Data(oldSp.AssetId.Bytes()).
				Data(oldSp.SourceID.Bytes())

			// spend input fields
			proof.Data(hashData(oldinp.ReferenceData).Bytes())

			// prevout id + data => vm1value + condition
			proof.Op(txvm.VM1Unlock)
		}
	}

	proof.Int64(int64(len(oldTx.Inputs))).Op(txvm.VM1Mux)

	// loop in reverse so that output 0 is at the top
	for i := len(oldTx.Outputs) - 1; i >= 0; i-- {
		oldout := oldTx.Outputs[i]
		proof.
			Int64(int64(oldout.Amount)).
			Data(oldout.AssetId.Bytes()).
			Op(txvm.VM1Withdraw).
			Data(hashData(oldout.ReferenceData).Bytes())
		if isRetirement(oldout.ControlProgram) {
			proof.Op(txvm.Retire)
		} else {
			proof.Data(oldout.ControlProgram).Op(txvm.Lock) // retains output object for checkoutput
		}
	}

	tx.Proof = proof.Build()

	for i := len(argsProgs) - 1; i >= 0; i-- {
		tx.Proof = append(tx.Proof, argsProgs[i]...)
	}

	return tx
}

func isRetirement(prog []byte) bool {
	return len(prog) > 0 && prog[0] == byte(vm.OP_FAIL)
}
