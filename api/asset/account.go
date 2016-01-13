package asset

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
)

type AccountReserver struct {
	AccountID string
}

func (reserver *AccountReserver) Reserve(ctx context.Context, assetAmount *bc.AssetAmount, ttl time.Duration) (*ReserveResult, error) {
	utxodbSource := utxodb.Source{
		AssetID:   assetAmount.AssetID,
		Amount:    assetAmount.Amount,
		AccountID: reserver.AccountID,
	}
	utxodbSources := []utxodb.Source{utxodbSource}
	reserved, change, err := utxoDB.Reserve(ctx, utxodbSources, ttl)
	if err != nil {
		return nil, err
	}

	result := &ReserveResult{}
	for _, r := range reserved {
		txInput := &bc.TxInput{
			Previous: r.Outpoint,
		}

		templateInput := &Input{}
		addrInfo, err := appdb.AddrInfo(ctx, r.AccountID)
		if err != nil {
			return nil, errors.Wrap(err, "get addr info")
		}
		signers := hdkey.Derive(addrInfo.Keys, appdb.ReceiverPath(addrInfo, r.AddrIndex[:]))
		redeemScript, err := hdkey.RedeemScript(signers, addrInfo.SigsRequired)
		if err != nil {
			return nil, errors.Wrap(err, "compute redeem script")
		}
		pkScript, err := hdkey.PayToRedeem(redeemScript)
		if err != nil {
			return nil, errors.Wrap(err, "compute pk script")
		}
		templateInput.RedeemScript = txscript.AddDataToScript(nil, redeemScript)
		templateInput.SignScript = pkScript
		templateInput.Sigs = inputSigs(signers)

		item := &ReserveResultItem{
			TxInput:       txInput,
			TemplateInput: templateInput,
		}

		result.Items = append(result.Items, item)
	}
	if len(change) > 0 {
		changeAssetAmount := &bc.AssetAmount{
			AssetID: assetAmount.AssetID,
			Amount:  change[0].Amount,
		}
		result.Change, err = NewAccountDestination(ctx, changeAssetAmount, reserver.AccountID, true, nil)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func NewAccountSource(ctx context.Context, assetAmount *bc.AssetAmount, accountID string) *Source {
	return &Source{
		AssetAmount: *assetAmount,
		Reserver: &AccountReserver{
			AccountID: accountID,
		},
	}
}

type AccountReceiver struct {
	addr *appdb.Address
}

func (receiver *AccountReceiver) IsChange() bool   { return receiver.addr.IsChange }
func (receiver *AccountReceiver) PKScript() []byte { return receiver.addr.PKScript }
func (receiver *AccountReceiver) AccumulateUTXO(ctx context.Context, outpoint *bc.Outpoint, txOutput *bc.TxOutput, utxoInserters []UTXOInserter) ([]UTXOInserter, error) {
	// Find or create an item in utxoInserters that is an
	// AccountUTXOInserter
	var accountUTXOInserter *AccountUTXOInserter
	for _, inserter := range utxoInserters {
		var ok bool
		if accountUTXOInserter, ok = inserter.(*AccountUTXOInserter); ok {
			break
		}
	}
	if accountUTXOInserter == nil {
		accountUTXOInserter = &AccountUTXOInserter{}
		utxoInserters = append(utxoInserters, accountUTXOInserter)
	}
	accountUTXOInserter.Add(outpoint, txOutput, receiver)
	return utxoInserters, nil
}

func (receiver *AccountReceiver) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"manager_node_id": receiver.addr.ManagerNodeID,
		"account_id":      receiver.addr.AccountID,
		"address_index":   receiver.addr.Index,
		"is_change":       receiver.addr.IsChange,
		"type":            "account",
	})
}

func NewAccountReceiver(addr *appdb.Address) *AccountReceiver {
	return &AccountReceiver{addr: addr}
}

func NewAccountDestination(ctx context.Context, assetAmount *bc.AssetAmount, accountID string, isChange bool, metadata []byte) (*Destination, error) {
	addr, err := appdb.NewAddress(ctx, accountID, isChange, false)
	if err != nil {
		return nil, err
	}
	receiver := NewAccountReceiver(addr)
	result := &Destination{
		AssetAmount: *assetAmount,
		IsChange:    isChange,
		Metadata:    metadata,
		Receiver:    receiver,
	}
	return result, nil
}

type AccountUTXOInserter struct {
	txdbOutputs []*txdb.Output
}

func (inserter *AccountUTXOInserter) Add(outpoint *bc.Outpoint, txOutput *bc.TxOutput, receiver *AccountReceiver) {
	txdbOutput := &txdb.Output{
		Output: state.Output{
			TxOutput: *txOutput,
			Outpoint: *outpoint,
		},
		ManagerNodeID: receiver.addr.ManagerNodeID,
		AccountID:     receiver.addr.AccountID,
	}
	copy(txdbOutput.AddrIndex[:], receiver.addr.Index)
	inserter.txdbOutputs = append(inserter.txdbOutputs, txdbOutput)
}

func (inserter *AccountUTXOInserter) InsertUTXOs(ctx context.Context) ([]*txdb.Output, error) {
	return inserter.txdbOutputs, txdb.InsertPoolOutputs(ctx, inserter.txdbOutputs)
}
