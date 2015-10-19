package asset

import (
	"bytes"
	"sort"
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/metrics"
)

// ErrPastExpires is returned by CreateAddress
// if the expiration time is in the past.
var ErrPastExpires = errors.New("expires in the past")

// CreateAddress uses appdb to allocate an address index for addr
// and insert it into the database.
// Fields BucketID, Amount, Expires, and IsChange must be set;
// all other fields will be initialized by CreateAddress.
// If Expires is not the zero time, but in the past,
// it returns ErrPastExpires.
func CreateAddress(ctx context.Context, addr *appdb.Address) error {
	defer metrics.RecordElapsed(time.Now())

	if !addr.Expires.IsZero() && addr.Expires.Before(time.Now()) {
		return errors.WithDetailf(ErrPastExpires, "%s ago", time.Since(addr.Expires))
	}

	err := addr.LoadNextIndex(ctx) // get most fields from the db given BucketID
	if err != nil {
		return errors.Wrap(err, "load")
	}
	signers := Signers(addr.Keys, ReceiverPath(addr))
	addr.RedeemScript, err = redeemScript(signers, addr.SigsRequired)
	if err != nil {
		return errors.Wrap(err, "compute redeem script")
	}
	bcAddr, err := btcutil.NewAddressScriptHash(addr.RedeemScript, &chaincfg.MainNetParams)
	if err != nil {
		return errors.Wrap(err, "compute address")
	}
	addr.PKScript, err = txscript.PayToAddrScript(bcAddr)
	if err != nil {
		return errors.Wrap(err, "compute pk script")
	}
	addr.Address = bcAddr.String()
	err = addr.Insert(ctx) // sets ID and Created
	return errors.Wrap(err, "save")
}

// Signers derives a key for each item in keys, according to path.
// The returned keys will be sorted by address.
func Signers(keys []*hdkey.XKey, path []uint32) []*DerivedKey {
	var a []*DerivedKey
	for _, k := range keys {
		a = append(a, &DerivedKey{k, path, addrPubKey(k, path)})
	}
	sort.Sort(byPubKey(a))
	return a
}

// Address returns the redeem script for the given signer set
// and number of required signatures.
func redeemScript(signers []*DerivedKey, nSigReq int) ([]byte, error) {
	var addrs []*btcutil.AddressPubKey
	for _, dk := range signers {
		addrs = append(addrs, dk.Address)
	}
	return txscript.MultiSigScript(addrs, nSigReq)
}

// DerivedKey represents an EC key derived from an xpub
// and derivation path.
// TODO(kr): rename hdkey.XKey to appdb.XKey and DerivedKey to Key.
type DerivedKey struct {
	Root    *hdkey.XKey
	Path    []uint32
	Address *btcutil.AddressPubKey
}

func addrPubKey(xkey *hdkey.XKey, path []uint32) *btcutil.AddressPubKey {
	// The only error has a uniformly distributed probability of 1/2^127
	// We've decided to ignore this chance.
	key := &xkey.ExtendedKey
	for _, p := range path {
		key, _ = key.Child(p)
	}
	eckey, _ := key.ECPubKey()
	addr, _ := btcutil.NewAddressPubKey(eckey.SerializeCompressed(), &chaincfg.MainNetParams)
	return addr
}

type byPubKey []*DerivedKey

func (a byPubKey) Len() int      { return len(a) }
func (a byPubKey) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPubKey) Less(i, j int) bool {
	ai := a[i].Address.ScriptAddress()
	aj := a[j].Address.ScriptAddress()
	return bytes.Compare(ai, aj) < 0
}
