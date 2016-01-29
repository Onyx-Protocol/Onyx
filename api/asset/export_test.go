package asset

import (
	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"

	"github.com/btcsuite/btcd/btcec"
)

func (ar *AccountReceiver) Addr() *appdb.Address {
	return ar.addr
}

func UTXODB() *utxodb.Reserver {
	return utxoDB
}

func NewKey() (pub, priv *hdkey.XKey, err error) {
	return newKey()
}

func Issued(outs []*bc.TxOutput) (bc.AssetID, uint64) {
	return issued(outs)
}

func IsSignedByTrustedHost(block *bc.Block, trustedKeys []*btcec.PublicKey) bool {
	return isSignedByTrustedHost(block, trustedKeys)
}

func GenerateBlockScript(keys []*btcec.PublicKey, nSigs int) ([]byte, error) {
	return generateBlockScript(keys, nSigs)
}
