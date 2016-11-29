package prottest

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519/chainkd"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
	"chain/testutil"
)

// NewIssuanceTx creates a new signed, issuance transaction issuing 100 units
// of a new asset to a garbage control program. The resulting transaction has
// one input and one output.
//
// The asset issued is created from randomly-generated keys. The resulting
// transaction is finalized (signed with a TXSIGHASH commitment).
func NewIssuanceTx(tb testing.TB, c *protocol.Chain) *bc.Tx {
	ctx := context.Background()
	b1, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	// Generate a random key pair for the asset being issued.
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	pubkeys := chainkd.XPubKeys([]chainkd.XPub{xpub})

	// Create a corresponding issuance program.
	sigProg, err := vmutil.P2SPMultiSigProgram(pubkeys, 1)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	builder := vmutil.NewBuilder()
	builder.AddData([]byte(`{"type": "prottest issuance"}`)).AddOp(vm.OP_DROP)
	builder.AddRawBytes(sigProg)
	issuanceProgram := builder.Program

	// Create a transaction issuing this new asset.
	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	txin := bc.NewIssuanceInput(nonce[:], 100, nil, b1.Hash(), issuanceProgram, nil)

	tx := bc.TxData{
		Version: bc.CurrentTransactionVersion,
		MinTime: bc.Millis(time.Now().Add(-5 * time.Minute)),
		MaxTime: bc.Millis(time.Now().Add(5 * time.Minute)),
		Inputs:  []*bc.TxInput{txin},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(txin.AssetID(), 100, []byte{0xbe, 0xef}, nil),
		},
	}

	// Sign with a simple TXSIGHASH signature.
	txhash := tx.HashForSig(0)
	builder = vmutil.NewBuilder()
	builder.AddData(txhash[:])
	builder.AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
	sigprog := builder.Program
	sigproghash := sha3.Sum256(sigprog)
	signature := xprv.Sign(sigproghash[:])

	var witness [][]byte
	witness = append(witness, vm.Int64Bytes(0)) // 0 args to the sigprog
	witness = append(witness, signature)
	witness = append(witness, sigprog)
	tx.Inputs[0].SetArguments(witness)

	return &bc.Tx{TxData: tx}
}
