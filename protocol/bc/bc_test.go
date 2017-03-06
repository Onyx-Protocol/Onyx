package bc

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"chain/errors"
	"chain/protocol/vm"
	"chain/testutil"
)

func serialize(t *testing.T, wt io.WriterTo) []byte {
	var b bytes.Buffer
	_, err := wt.WriteTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func mustDecodeHash(hash string) (h [32]byte) {
	if len(hash) != hex.EncodedLen(len(h)) {
		panic("wrong length hash")
	}
	_, err := hex.Decode(h[:], []byte(hash))
	if err != nil {
		panic(err)
	}
	return h
}

type txFixture struct {
	initialBlockID       Hash
	issuanceProg         Program
	issuanceArgs         [][]byte
	assetDef             []byte
	assetID              AssetID
	txVersion            uint64
	txInputs             []*TxInput
	txOutputs            []*TxOutput
	txMinTime, txMaxTime uint64
	txRefData            []byte
	tx                   *TxData
}

func sample(tb testing.TB, in *txFixture) *txFixture {
	var result txFixture
	if in != nil {
		result = *in
	}

	if (result.initialBlockID == Hash{}) {
		result.initialBlockID = Hash{1}
	}
	if testutil.DeepEqual(result.issuanceProg, Program{}) {
		prog, err := vm.Assemble("2 3 ADD NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		result.issuanceProg = Program{VMVersion: 1, Code: prog}
	}
	if len(result.issuanceArgs) == 0 {
		result.issuanceArgs = [][]byte{[]byte{5}}
	}
	if len(result.assetDef) == 0 {
		result.assetDef = []byte{2}
	}
	if (result.assetID == AssetID{}) {
		result.assetID = ComputeAssetID(result.issuanceProg.Code, result.initialBlockID, result.issuanceProg.VMVersion, hashData(result.assetDef))
	}

	if result.txVersion == 0 {
		result.txVersion = 1
	}
	if len(result.txInputs) == 0 {
		cp1, err := vm.Assemble("4 5 ADD NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args1 := [][]byte{[]byte{9}}

		cp2, err := vm.Assemble("6 7 ADD NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args2 := [][]byte{[]byte{13}}

		result.txInputs = []*TxInput{
			NewIssuanceInput([]byte{3}, 10, []byte{4}, result.initialBlockID, result.issuanceProg.Code, result.issuanceArgs, result.assetDef),
			NewSpendInput(args1, Hash{5}, result.assetID, 20, 0, cp1, Hash{6}, []byte{7}),
			NewSpendInput(args2, Hash{8}, result.assetID, 40, 0, cp2, Hash{9}, []byte{10}),
		}
	}
	if len(result.txOutputs) == 0 {
		cp1, err := vm.Assemble("8 9 ADD NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		cp2, err := vm.Assemble("10 11 ADD NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}

		result.txOutputs = []*TxOutput{
			NewTxOutput(result.assetID, 25, cp1, []byte{11}),
			NewTxOutput(result.assetID, 45, cp2, []byte{12}),
		}
	}
	if result.txMinTime == 0 {
		result.txMinTime = Millis(time.Now().Add(-time.Minute))
	}
	if result.txMaxTime == 0 {
		result.txMaxTime = Millis(time.Now().Add(time.Minute))
	}
	if len(result.txRefData) == 0 {
		result.txRefData = []byte{13}
	}

	result.tx = &TxData{
		Version:       result.txVersion,
		Inputs:        result.txInputs,
		Outputs:       result.txOutputs,
		MinTime:       result.txMinTime,
		MaxTime:       result.txMaxTime,
		ReferenceData: result.txRefData,
	}

	return &result
}

// Like errors.Root, but also unwraps vm.Error objects.
func rootErr(e error) error {
	for {
		e = errors.Root(e)
		if e2, ok := e.(vm.Error); ok {
			e = e2.Err
			continue
		}
		return e
	}
}
