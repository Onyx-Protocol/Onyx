package asset

import (
	"bytes"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
)

// ErrBadTx is returned by FinalizeTx
var ErrBadTx = errors.New("bad transaction template")

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, tx *Tx) (*wire.MsgTx, error) {
	defer metrics.RecordElapsed(time.Now())
	msg := wire.NewMsgTx()
	err := msg.Deserialize(bytes.NewReader(tx.Unsigned))
	if err != nil {
		return nil, errors.WithDetailf(ErrBadTx, "invalid unsigned transaction hex")
	}

	if len(tx.Inputs) > len(msg.TxIn) {
		return nil, errors.WithDetail(ErrBadTx, "too many inputs in template")
	}

	// TODO(erykwalder): make sure n signatures are valid
	// for input, once more than 1-of-1 is supported.
	for i, input := range tx.Inputs {
		if len(input.Sigs) == 0 {
			return nil, errors.WithDetailf(ErrBadTx, "input %d must contain signatures", i)
		}
		for j, sig := range input.Sigs {
			key, err := hdkey.NewXKey(sig.XPub)
			if err != nil {
				return nil, errors.WithDetailf(ErrBadTx, "invalid xpub for input %d signature %d", i, j)
			}

			addr := addrPubKey(key, sig.DerivationPath)
			err = checkSig(addr.PubKey(), input.SignatureData, sig.DER)

			if err != nil {
				return nil, errors.WithDetailf(ErrBadTx, "error for input %d signature %d: %v", i, j, err)
			}

			msg.TxIn[i].SignatureScript = append(msg.TxIn[i].SignatureScript, sig.DER...)
		}
		msg.TxIn[i].SignatureScript = append(msg.TxIn[i].SignatureScript, input.RedeemScript...)
	}

	err = utxoDB.Apply(ctx, msg)
	if err != nil {
		return nil, errors.Wrap(err, "storing txn")
	}

	if isIssuance(msg) {
		asset, amt := issued(msg.TxOut)
		err = appdb.AddIssuance(ctx, asset.String(), amt)
		if err != nil {
			return nil, errors.Wrap(err, "writing issued assets")
		}
	}

	return msg, nil
}

func checkSig(key *btcec.PublicKey, data, sig []byte) error {
	ecSig, err := btcec.ParseDERSignature(sig, btcec.S256())
	if err != nil {
		return errors.Wrapf(err, "invalid der signature %x", sig)
	}

	if !ecSig.Verify(data, key) {
		return errors.Wrap(fmt.Errorf("signature %x not valid for pubkey", sig))
	}

	return nil
}

func isIssuance(msg *wire.MsgTx) bool {
	emptyHash := wire.Hash32{}
	if len(msg.TxIn) == 1 && msg.TxIn[0].PreviousOutPoint.Hash == emptyHash {
		if len(msg.TxOut) == 0 {
			return false
		}
		assetID := msg.TxOut[0].AssetID
		for _, out := range msg.TxOut {
			if out.AssetID != assetID {
				return false
			}
		}
		return true
	}
	return false
}

// issued returns the asset issued, as well as the amount.
// It should only be called with outputs from transactions
// where isIssuance is true.
func issued(outs []*wire.TxOut) (asset wire.Hash20, amt int64) {
	for _, out := range outs {
		amt += out.Value
	}
	return outs[0].AssetID, amt
}
