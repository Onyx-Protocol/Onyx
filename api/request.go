package api

import (
	"errors"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	chainjson "chain/encoding/json"
	"chain/fedchain/bc"
)

// Data types and functions for marshaling/unmarshaling API requests

var (
	ErrNullAsset              = errors.New("asset type unspecified")
	ErrUnknownReceiverType    = errors.New("unknown request type")
	ErrUnknownSourceType      = errors.New("unknown source type")
	ErrUnknownDestinationType = errors.New("unknown destination type")
)

type Source struct {
	AssetID   *bc.AssetID `json:"asset_id"`
	Amount    uint64
	AccountID string `json:"account_id"`
	Type      string
}

func (source *Source) parse(ctx context.Context) (*asset.Source, error) {
	if source.Type == "account" || source.Type == "" {
		if source.AssetID == nil {
			return nil, ErrNullAsset
		}
		assetAmount := &bc.AssetAmount{
			AssetID: *source.AssetID,
			Amount:  source.Amount,
		}
		return asset.NewAccountSource(ctx, assetAmount, source.AccountID), nil
	}
	return nil, ErrUnknownSourceType
}

type Destination struct {
	AssetID   *bc.AssetID `json:"asset_id"`
	Amount    uint64
	AccountID string `json:"account_id"`
	Address   chainjson.HexBytes
	IsChange  bool `json:"is_change"`
	Metadata  chainjson.HexBytes
	Type      string
}

func (dest Destination) parse(ctx context.Context) (*asset.Destination, error) {
	if dest.AssetID == nil {
		return nil, ErrNullAsset
	}

	// backwards compatibility fix
	if dest.Type == "" && len(dest.Address) != 0 {
		dest.Type = "address"
	}

	assetAmount := &bc.AssetAmount{
		AssetID: *dest.AssetID,
		Amount:  dest.Amount,
	}

	switch dest.Type {
	case "account", "":
		return asset.NewAccountDestination(ctx, assetAmount, dest.AccountID, dest.IsChange, dest.Metadata)
	case "address":
		return asset.NewScriptDestination(ctx, assetAmount, dest.Address, dest.IsChange, dest.Metadata)
	}
	return nil, ErrUnknownDestinationType
}

type Receiver struct {
	AccountID     string   `json:"account_id"`
	AddrIndex     []uint32 `json:"address_index"`
	IsChange      bool     `json:"is_change"`
	ManagerNodeID string   `json:"manager_node_id"`
	Script        chainjson.HexBytes
	Type          string
}

func (receiver *Receiver) parse() (asset.Receiver, error) {
	// backwards compatibility fix
	if receiver.Type == "" && receiver.AccountID != "" {
		receiver.Type = "account"
	}

	switch receiver.Type {
	case "script", "":
		return asset.NewScriptReceiver(receiver.Script, receiver.IsChange), nil
	case "account":
		addr := &appdb.Address{
			AccountID:     receiver.AccountID,
			Index:         receiver.AddrIndex,
			ManagerNodeID: receiver.ManagerNodeID,
		}
		return asset.NewAccountReceiver(addr), nil
	}
	return nil, ErrUnknownReceiverType
}

type Template struct {
	Unsigned   *bc.TxData `json:"unsigned_hex"`
	BlockChain string     `json:"block_chain"`
	Inputs     []*asset.Input
	OutRecvs   []Receiver `json:"output_receivers"`
}

func (tpl *Template) parse(ctx context.Context) (*asset.TxTemplate, error) {
	result := &asset.TxTemplate{
		Unsigned:   tpl.Unsigned,
		BlockChain: tpl.BlockChain,
		Inputs:     tpl.Inputs,
	}
	for _, receiver := range tpl.OutRecvs {
		parsed, err := receiver.parse()
		if err != nil {
			return nil, err
		}
		result.OutRecvs = append(result.OutRecvs, parsed)
	}
	return result, nil
}

type BuildRequest struct {
	PrevTx  *Template      `json:"previous_transaction"`
	Sources []*Source      `json:"inputs"`
	Dests   []*Destination `json:"outputs"`
	ResTime time.Duration  `json:"reservation_duration"`
}

func (req *BuildRequest) parse(ctx context.Context) (*asset.TxTemplate, []*asset.Source, []*asset.Destination, error) {
	var (
		prevTx *asset.TxTemplate
		err    error
	)
	if req.PrevTx != nil {
		prevTx, err = req.PrevTx.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	sources := make([]*asset.Source, 0, len(req.Sources))
	destinations := make([]*asset.Destination, 0, len(req.Dests))

	for _, source := range req.Sources {
		parsed, err := source.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		sources = append(sources, parsed)
	}
	for _, destination := range req.Dests {
		parsed, err := destination.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		destinations = append(destinations, parsed)
	}
	return prevTx, sources, destinations, nil
}
