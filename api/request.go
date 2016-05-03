package api

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/api/issuer"
	"chain/api/smartcontracts/orderbook"
	"chain/api/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
)

// Data types and functions for marshaling/unmarshaling API requests

type Source struct {
	AssetID        *bc.AssetID `json:"asset_id"`
	Amount         uint64
	PaymentAssetID *bc.AssetID `json:"payment_asset_id"`
	PaymentAmount  uint64      `json:"payment_amount"`
	AccountID      string      `json:"account_id"`
	TxHash         *bc.Hash    `json:"transaction_hash"`
	Index          *uint32     `json:"index"`
	Type           string
	// ClientToken is an idempotency key to guarantee one-time reservation.
	ClientToken *string `json:"client_token"`

	// TxHashAsID exists only to provide an alternate input alias
	// ("transaction_id") for TxHash. This field should be treated as read-only.
	TxHashAsID *bc.Hash `json:"transaction_id"`
}

func (source *Source) parse(ctx context.Context) (*txbuilder.Source, error) {
	// source.TxHash can be provided via JSON as either "transaction_hash" or
	// "transaction_id". Each JSON key has its own struct field, but only
	// source.TxHash should be used outside of this function.
	if source.TxHash != nil && source.TxHashAsID != nil {
		return nil, errors.WithDetail(ErrBadBuildRequest, "transaction_id and transaction_hash are both specified, please use transaction_id only")
	}
	if source.TxHash == nil {
		source.TxHash = source.TxHashAsID
	}
	source.TxHashAsID = nil
	switch source.Type {
	case "account", "":
		if source.AccountID == "" {
			return nil, errors.WithDetail(ErrBadBuildRequest, "account_id is not specified on the input")
		}
		if source.AssetID == nil {
			return nil, errors.WithDetail(ErrBadBuildRequest, "asset_id is not specified on the input")
		}
		if source.Amount == 0 {
			return nil, errors.WithDetailf(ErrBadBuildRequest,
				"input for asset %s has zero amount", source.AssetID)
		}
		assetAmount := &bc.AssetAmount{
			AssetID: *source.AssetID,
			Amount:  source.Amount,
		}
		return asset.NewAccountSource(ctx, assetAmount, source.AccountID, source.TxHash, source.ClientToken), nil
	case "issue":
		if source.AssetID == nil {
			return nil, errors.WithDetail(ErrBadBuildRequest, "asset_id is not specified on the issuance input")
		}
		assetAmount := &bc.AssetAmount{
			AssetID: *source.AssetID,
			Amount:  source.Amount,
		}
		return issuer.NewIssueSource(ctx, assetAmount), nil
	case "orderbook-redeem":
		if source.PaymentAssetID == nil {
			return nil, errors.WithDetail(ErrBadBuildRequest, "asset_id is not specified on the orderbook-redeem input")
		}
		if source.PaymentAmount == 0 {
			return nil, errors.WithDetailf(ErrBadBuildRequest,
				"input for asset %s has zero amount", *source.PaymentAssetID)
		}
		if source.TxHash == nil {
			return nil, errors.WithDetailf(ErrBadBuildRequest, "transaction_id is not specified on the orderbook-redeem input")
		}
		if source.Index == nil {
			return nil, errors.WithDetailf(ErrBadBuildRequest, "index is not specified on the orderbook-redeem input")
		}
		outpoint := &bc.Outpoint{
			Hash:  *source.TxHash,
			Index: *source.Index,
		}
		openOrder, err := orderbook.FindOpenOrderByOutpoint(ctx, outpoint)
		if err != nil {
			return nil, err
		}
		if openOrder == nil {
			return nil, pg.ErrUserInputNotFound
		}
		paymentAmount := &bc.AssetAmount{
			AssetID: *source.PaymentAssetID,
			Amount:  source.PaymentAmount,
		}
		return orderbook.NewRedeemSource(openOrder, source.Amount, paymentAmount), nil
	case "orderbook-cancel":
		if source.TxHash == nil {
			return nil, errors.WithDetailf(ErrBadBuildRequest, "transaction_id is not specified on the orderbook-cancel input")
		}
		if source.Index == nil {
			return nil, errors.WithDetailf(ErrBadBuildRequest, "index is not specified on the orderbook-cancel input")
		}
		outpoint := &bc.Outpoint{
			Hash:  *source.TxHash,
			Index: *source.Index,
		}
		openOrder, err := orderbook.FindOpenOrderByOutpoint(ctx, outpoint)
		if err != nil {
			return nil, err
		}
		if openOrder == nil {
			return nil, pg.ErrUserInputNotFound
		}
		return orderbook.NewCancelSource(openOrder), nil
	}
	return nil, errors.WithDetailf(ErrBadBuildRequest, "unknown source type `%s`", source.Type)
}

type Destination struct {
	AssetID         *bc.AssetID `json:"asset_id"`
	Amount          uint64
	AccountID       string             `json:"account_id,omitempty"`
	Address         chainjson.HexBytes `json:"address,omitempty"`
	Metadata        chainjson.HexBytes `json:"metadata,omitempty"`
	OrderbookPrices []*orderbook.Price `json:"orderbook_prices,omitempty"`
	Script          chainjson.HexBytes `json:"script,omitempty"`
	Type            string
}

// buildAddress will return the destination's script, if populated. Otherwise,
// it will create a new address for the destination's account ID.
func buildAddress(ctx context.Context, script []byte, accountID string) ([]byte, error) {
	if script != nil {
		return script, nil
	}
	if accountID == "" {
		return nil, errors.WithDetailf(ErrBadBuildRequest, "need either script or account id")
	}
	addr, err := appdb.NewAddress(ctx, accountID, true)
	if err != nil {
		return nil, errors.Wrapf(err, "generating address, accountID %s", accountID)
	}
	return addr.PKScript, nil
}

func (dest Destination) parse(ctx context.Context) (*txbuilder.Destination, error) {
	if dest.AssetID == nil {
		return nil, errors.WithDetail(ErrBadBuildRequest, "asset_id is not specified on output")
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
		if dest.AccountID == "" {
			return nil, errors.WithDetail(ErrBadBuildRequest, "account_id is not specified on output")
		}
		return asset.NewAccountDestination(ctx, assetAmount, dest.AccountID, dest.Metadata)
	case "address":
		return txbuilder.NewScriptDestination(ctx, assetAmount, dest.Address, dest.Metadata), nil
	case "retire":
		return txbuilder.NewRetireDestination(ctx, assetAmount, dest.Metadata), nil
	case "orderbook":
		orderInfo := &orderbook.OrderInfo{
			SellerAccountID: dest.AccountID,
			Prices:          dest.OrderbookPrices,
		}
		return orderbook.NewDestinationWithScript(ctx, assetAmount, orderInfo, dest.Metadata, dest.Script)
	}
	return nil, errors.WithDetailf(ErrBadBuildRequest, "unknown destination type `%s`", dest.Type)
}

type actionDetails struct {
	Contract    string
	Type        string
	Metadata    chainjson.HexBytes
	ClientToken *string
}

type Action struct {
	actionDetails
	Raw json.RawMessage
}

func (a *Action) UnmarshalJSON(b []byte) error {
	// Hold onto the original bytes for decoding action-specific fields.
	a.Raw = json.RawMessage(b)

	err := json.Unmarshal(b, &a.actionDetails)
	if err != nil {
		return err
	}
	return nil
}

func (a *Action) UnmarshalInto(v interface{}) error {
	return json.Unmarshal(a.Raw, v)
}

func (a *Action) parse(ctx context.Context) (srcs []*txbuilder.Source, dsts []*txbuilder.Destination, err error) {
	switch a.Contract {
	case "voting":
		srcs, dsts, err = parseVotingAction(ctx, a)
	default:
		err = errors.WithDetailf(ErrBadBuildRequest, "unknown contract `%s`", a.Contract)
	}
	return srcs, dsts, err
}

type BuildRequest struct {
	PrevTx   *txbuilder.Template `json:"previous_transaction"`
	Sources  []*Source           `json:"inputs"`
	Dests    []*Destination      `json:"outputs"`
	Actions  []*Action           `json:"actions"`
	Metadata chainjson.HexBytes  `json:"metadata"`
	ResTime  time.Duration       `json:"reservation_duration"`
}

func (req *BuildRequest) parse(ctx context.Context) (*txbuilder.Template, []*txbuilder.Source, []*txbuilder.Destination, error) {
	var (
		sources      = make([]*txbuilder.Source, 0, len(req.Sources))
		destinations = make([]*txbuilder.Destination, 0, len(req.Dests))
	)

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
	for _, action := range req.Actions {
		srcs, dsts, err := action.parse(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		sources = append(sources, srcs...)
		destinations = append(destinations, dsts...)
	}
	return req.PrevTx, sources, destinations, nil
}
