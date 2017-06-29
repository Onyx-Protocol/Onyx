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
func NewIssuanceTx(tb testing.TB, initial bc.Hash) []byte {
	// Generate a random key pair for the asset being issued.
	xprv, xpub, err := chainkd.NewXKeys(nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	issuanceProgram := fmt.Sprintf(`7 id encode [11 id cat sha3 %x checksig] cat defer`, xpub.PublicKey())

	// Create a transaction issuing this new asset.
	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	assetdef := []byte(`{"type": "prottest issuance"}`)
	nonceProgram := fmt.Sprintf(`%x drop`, nonce[:])
	mintime := bc.Millis(time.Now().Add(-5 * time.Minute))
	maxtime := bc.Millis(time.Now().Add(5 * time.Minute))

	tx, err := txvm.Assemble(fmt.Sprintf(`
		{'nonce', [%s], %d, %d} anchor
		100 {'assetdefinition', {"%x"x}, [%s]}
		[1 verify] 1 lock
		summarize
	`, nonceProgram, mintime, maxtime, assetdef, issuanceProgram))
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	deserialized, err := bcvm.NewTx(tx)
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	hasher.Write(deserialized.Nonces[0].ID.Bytes())
	hasher.Write(deserialized.ID.Bytes())

	var hash [32]byte
	hasher.Read(hash[:])

	sig, err := xprv.Sign(hash[:])

	return tx
}
