package validation

import (
	"testing"
	"time"

	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
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
	transactionsRoot := bc.NewHash([32]byte{1})
	b1.Body.TransactionsRoot = &transactionsRoot // make b1 be invalid
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
	transactionsRoot := bc.NewHash([32]byte{1})
	b2.Body.TransactionsRoot = &transactionsRoot // make b2 be invalid
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

func dummyValidateTx(*bc.Tx) error {
	return nil
}

func newInitialBlock(tb testing.TB) *bc.Block {
	script, err := vmutil.BlockMultiSigProgram(nil, 0)
	if err != nil {
		tb.Fatal(err)
	}

	root, err := bc.MerkleRoot(nil) // calculate the zero value of the tx merkle root
	if err != nil {
		tb.Fatal(err)
	}

	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:     1,
			Height:      1,
			TimestampMS: bc.Millis(time.Now()),
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: root,
				ConsensusProgram:       script,
			},
		},
	}
	return legacy.MapBlock(b)
}

func generate(tb testing.TB, prev *bc.Block) *bc.Block {
	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            prev.Body.Height + 1,
			PreviousBlockHash: prev.ID,
			TimestampMS:       prev.Body.TimestampMs + 1,
			BlockCommitment: legacy.BlockCommitment{
				ConsensusProgram: prev.Body.NextConsensusProgram,
			},
		},
	}

	var err error
	b.TransactionsMerkleRoot, err = bc.MerkleRoot(nil)
	if err != nil {
		tb.Fatal(err)
	}

	return legacy.MapBlock(b)
}
