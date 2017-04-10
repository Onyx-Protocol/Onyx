package validation

import (
	"testing"
	"time"

	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

func TestValidateBlock1(t *testing.T) {
	b1 := newInitialBlock(t)
	err := ValidateBlock(b1, nil, b1.ID, dummyValidateTx)
	if err != nil {
		t.Errorf("ValidateBlock(%v, nil) = %v, want nil", b1, err)
	}
}

func TestValidateBlock1Err(t *testing.T) {
	b1 := newInitialBlock(t)
	b1.Body.TransactionsRoot = bc.NewHash([32]byte{0x01}) // make b1 be invalid
	err := ValidateBlock(b1, nil, b1.ID, dummyValidateTx)
	if err == nil {
		t.Errorf("ValidateBlock(%v, nil) = nil, want error", b1)
	}
}

func TestValidateBlock2(t *testing.T) {
	b1 := newInitialBlock(t)
	b2 := generate(t, b1)
	err := ValidateBlock(b2, b1, b2.ID, dummyValidateTx)
	if err != nil {
		t.Errorf("ValidateBlock(%v, %v) = %v, want nil", b2, b1, err)
	}
}

func TestValidateBlock2Err(t *testing.T) {
	b1 := newInitialBlock(t)
	b2 := generate(t, b1)
	b2.Body.TransactionsRoot = bc.NewHash([32]byte{0x01}) // make b2 be invalid
	err := ValidateBlock(b2, b1, b2.ID, dummyValidateTx)
	if err == nil {
		t.Errorf("ValidateBlock(%v, %v) = nil, want error", b2, b1)
	}
}

func TestValidateBlockSig2(t *testing.T) {
	b1 := newInitialBlock(t)
	b2 := generate(t, b1)
	err := ValidateBlockSig(b2, b1.Body.NextConsensusProgram)
	if err != nil {
		t.Errorf("ValidateBlockSig(%v, %v) = %v, want nil", b2, b1, err)
	}
}

func TestValidateBlockSig2Err(t *testing.T) {
	b1 := newInitialBlock(t)
	b2 := generate(t, b1)
	prog := []byte{byte(vm.OP_FALSE)} // make b2 be invalid
	err := ValidateBlockSig(b2, prog)
	if err == nil {
		t.Errorf("ValidateBlockSig(%v, %v) = nil, want error", b2, b1)
	}
}

func dummyValidateTx(*bc.TxEntries) error {
	return nil
}

func newInitialBlock(tb testing.TB) *bc.BlockEntries {
	script, err := vmutil.BlockMultiSigProgram(nil, 0)
	if err != nil {
		tb.Fatal(err)
	}

	root, err := bc.MerkleRoot(nil) // calculate the zero value of the tx merkle root
	if err != nil {
		tb.Fatal(err)
	}

	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:     bc.NewBlockVersion,
			Height:      1,
			TimestampMS: bc.Millis(time.Now()),
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: root,
				ConsensusProgram:       script,
			},
		},
	}
	return bc.MapBlock(b)
}

func generate(tb testing.TB, prev *bc.BlockEntries) *bc.BlockEntries {
	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           bc.NewBlockVersion,
			Height:            prev.Body.Height + 1,
			PreviousBlockHash: prev.ID,
			TimestampMS:       prev.Body.TimestampMS + 1,
			BlockCommitment: bc.BlockCommitment{
				ConsensusProgram: prev.Body.NextConsensusProgram,
			},
		},
	}

	var err error
	b.TransactionsMerkleRoot, err = bc.MerkleRoot(nil)
	if err != nil {
		tb.Fatal(err)
	}

	return bc.MapBlock(b)
}
