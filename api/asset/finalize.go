package asset

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
	"chain/strings"
)

// ErrBadTx is returned by FinalizeTx
var ErrBadTx = errors.New("bad transaction template")

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, tx *Tx) (*wire.MsgTx, error) {
	msg := wire.NewMsgTx()
	err := msg.Deserialize(bytes.NewReader(tx.Unsigned))
	if err != nil {
		return nil, errors.WithDetailf(ErrBadTx, "invalid unsigned transaction hex")
	}

	var keyIDs []string
	for _, input := range tx.Inputs {
		for _, sig := range input.Sigs {
			keyIDs = append(keyIDs, sig.XPubHash)
		}
	}
	sort.Strings(keyIDs)
	keyIDs = strings.Uniq(keyIDs)

	keys, err := appdb.GetKeys(ctx, keyIDs)
	if err == appdb.ErrMissingKeys {
		return nil, errors.WithDetailf(ErrBadTx, "could not find all keys in template")
	} else if err != nil {
		return nil, errors.Wrap(err)
	}

	keyMap := make(map[string]*appdb.Key)
	for _, k := range keys {
		keyMap[k.ID] = k
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
			key := keyMap[sig.XPubHash]
			addr := addrPubKey(key, sig.DerivationPath)
			err := checkSig(addr.PubKey(), input.SignatureData, sig.DER)

			if err != nil {
				return nil, errors.WithDetailf(ErrBadTx, "error for input %d signature %d: %v", i, j, err)
			}

			msg.TxIn[i].SignatureScript = append(msg.TxIn[i].SignatureScript, sig.DER...)
		}
		msg.TxIn[i].SignatureScript = append(msg.TxIn[i].SignatureScript, input.RedeemScript...)
	}

	err = appdb.Commit(ctx, msg)
	if err != nil {
		return nil, errors.Wrap(err, "storing txn")
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
