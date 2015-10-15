package asset

import (
	"time"

	"golang.org/x/net/context"

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
// If save is false, it will skip saving the address;
// in that case ID will remain unset.
// If Expires is not the zero time, but in the past,
// it returns ErrPastExpires.
func CreateAddress(ctx context.Context, addr *appdb.Address, save bool) error {
	defer metrics.RecordElapsed(time.Now())

	if !addr.Expires.IsZero() && addr.Expires.Before(time.Now()) {
		return errors.WithDetailf(ErrPastExpires, "%s ago", time.Since(addr.Expires))
	}

	err := addr.LoadNextIndex(ctx) // get most fields from the db given BucketID
	if err != nil {
		return errors.Wrap(err, "load")
	}

	var bcAddr *btcutil.AddressScriptHash
	bcAddr, addr.RedeemScript, err = hdkey.Address(addr.Keys, appdb.ReceiverPath(addr, addr.Index), addr.SigsRequired)
	if err != nil {
		return errors.Wrap(err, "compute redeem script")
	}

	addr.PKScript, err = txscript.PayToAddrScript(bcAddr)
	if err != nil {
		return errors.Wrap(err, "compute pk script")
	}
	addr.Address = bcAddr.String()
	if !save {
		addr.Created = time.Now()
		return nil
	}
	err = addr.Insert(ctx) // sets ID and Created
	return errors.Wrap(err, "save")
}
