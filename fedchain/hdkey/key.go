package hdkey

import (
	"bytes"
	"sort"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"

	"chain/crypto/hash160"
	"chain/errors"
	"chain/fedchain/txscript"
)

// XKey represents an extended key,
// with additional methods to marshal and unmarshal as text,
// for JSON encoding.
// The embedded type carries methods with it;
// see its documentation for details.
type XKey struct {
	hdkeychain.ExtendedKey
}

// Key represents an EC key derived from an xpub
// and derivation path.
type Key struct {
	Root    *XKey
	Path    []uint32
	Address *btcutil.AddressPubKey
}

// New returns a new public/private XKey pair.
func New() (pub, priv *XKey, err error) {
	seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating key seed")
	}
	xprv, err := hdkeychain.NewMaster(seed)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating root xprv")
	}
	xpub, err := xprv.Neuter()
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting root xpub")
	}
	return &XKey{ExtendedKey: *xpub}, &XKey{ExtendedKey: *xprv}, nil
}

func (k XKey) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *XKey) UnmarshalText(p []byte) error {
	key, err := hdkeychain.NewKeyFromString(string(p))
	if err != nil {
		return errors.Wrap(err, "unmarshal XKey")
	}
	k.ExtendedKey = *key
	return nil
}

func NewXKey(pubstr string) (*XKey, error) {
	extkey, err := hdkeychain.NewKeyFromString(pubstr)
	if err != nil {
		return nil, err
	}
	return &XKey{ExtendedKey: *extkey}, nil
}

// RedeemScript returns the redeem script
// for the given set of signers
// and number of required signatures.
func RedeemScript(signers []*Key, nSigReq int) ([]byte, error) {
	var addrs []*btcutil.AddressPubKey
	for _, key := range signers {
		addrs = append(addrs, key.Address)
	}
	return txscript.MultiSigScript(addrs, nSigReq)
}

// Scripts computes the P2SH redeem script
// and corresponding pk script
// for the given set of keys and derivation path.
func Scripts(xkeys []*XKey, path []uint32, nSigReq int) (pkScript, redeemScript []byte, err error) {
	redeemScript, err = RedeemScript(Derive(xkeys, path), nSigReq)
	if err != nil {
		return nil, nil, errors.Wrap(err, "compute redeem script")
	}

	pkScript, err = PayToRedeem(redeemScript)
	if err != nil {
		return nil, nil, err
	}

	return pkScript, redeemScript, nil
}

// PayToRedeem takes a redeem script
// and calculates its corresponding pk script
func PayToRedeem(redeem []byte) ([]byte, error) {
	hash := hash160.Sum(redeem)
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_DUP)
	builder.AddOp(txscript.OP_HASH160)
	builder.AddData(hash[:])
	builder.AddOp(txscript.OP_EQUALVERIFY)
	builder.AddOp(txscript.OP_EVAL)
	return builder.Script()
}

// Derive derives a key for each item in xkeys, according to path.
// The returned xkeys will be sorted by address.
func Derive(xkeys []*XKey, path []uint32) []*Key {
	var a []*Key
	for _, xkey := range xkeys {
		a = append(a, &Key{xkey, path, DeriveAPK(xkey, path)})
	}
	sort.Sort(byPubKey(a))
	return a
}

func DeriveAPK(xkey *XKey, path []uint32) *btcutil.AddressPubKey {
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

type byPubKey []*Key

func (a byPubKey) Len() int      { return len(a) }
func (a byPubKey) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPubKey) Less(i, j int) bool {
	ai := a[i].Address.ScriptAddress()
	aj := a[j].Address.ScriptAddress()
	return bytes.Compare(ai, aj) < 0
}
