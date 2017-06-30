// Package bcvmtest provides utilities for constructing blockchain data
// structures.
package bcvmtest

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"chain/crypto/ed25519/chainkd"
	"chain/crypto/sha3pool"
	"chain/protocol/bc"
	"chain/protocol/bc/bcvm"
	"chain/protocol/txvm"
	"chain/testutil"
)

// NewIssuanceTx creates a new signed, issuance transaction issuing 100 units
// of a new asset to a garbage control program. The resulting transaction has
// one input and one output.
//
// The asset issued is created from randomly-generated keys. The resulting
// transaction is finalized (signed with a TXSIGHASH commitment).
func NewIssuanceTx(tb testing.TB) []byte {
	// Generate a random key pair for the asset being issued.
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	issuanceProgram := fmt.Sprintf(`7 id encode [11 id cat sha3 "%x"x checksig] cat defer`, xpub.PublicKey())

	// Create a transaction issuing this new asset.
	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	nonceProgram := fmt.Sprintf(`"%x"x drop`, nonce[:])
	mintime := bc.Millis(time.Now().Add(-5 * time.Minute))
	maxtime := bc.Millis(time.Now().Add(5 * time.Minute))

	tx, err := txvm.Assemble(fmt.Sprintf(`
		{'nonce', [%s], %d, %d} nonce
		100 {'assetdefinition', [%s]} issue
		[1 verify] 1 lock
		summarize
	`, nonceProgram, mintime, maxtime, issuanceProgram))
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	deserialized, _ := bcvm.NewTx(tx)

	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	hasher.Write(deserialized.IssueAnchors[0].Bytes())
	hasher.Write(deserialized.ID.Bytes())

	var hash [32]byte
	hasher.Read(hash[:])

	sig := xprv.Sign(hash[:])

	return append(tx, (&txvm.Builder{}).Data(sig).Op(txvm.Satisfy).Build()...)
}
