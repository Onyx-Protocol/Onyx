package legacy_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/protocol/bc"
	"chain/protocol/bc/bctest"
	"chain/protocol/bc/legacy"
)

func TestMapVmTx(t *testing.T) {
	issuance := bctest.NewIssuanceTx(t, bc.Hash{})

	result := legacy.MapVMTx(&issuance.TxData)
	t.Log(spew.Sdump(result))
	t.Logf("%x\n", result.Proof)
	t.Fatal()
}
