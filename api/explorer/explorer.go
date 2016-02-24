package explorer

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
)

// ListBlocksItem is returned by ListBlocks
type ListBlocksItem struct {
	ID      bc.Hash   `json:"id"`
	Height  uint64    `json:"height"`
	Time    time.Time `json:"time"`
	TxCount int       `json:"transaction_count"`
}

// ListBlocks returns an array of ListBlocksItems
// as well as a pagination pointer for the last item
// in the list.
func ListBlocks(ctx context.Context, prev string, limit int) ([]ListBlocksItem, string, error) {
	blocks, err := txdb.ListBlocks(ctx, prev, limit)
	if err != nil {
		return nil, "", err
	}

	var (
		list []ListBlocksItem
		last string
	)
	for _, b := range blocks {
		list = append(list, ListBlocksItem{b.Hash(), b.Height, b.Time(), len(b.Transactions)})
	}
	if len(list) == limit && limit > 0 {
		last = fmt.Sprintf("%d", list[len(list)-1].Height)
	}

	return list, last, nil
}

// BlockSummary is returned by GetBlockSummary
type BlockSummary struct {
	ID      bc.Hash   `json:"id"`
	Height  uint64    `json:"height"`
	Time    time.Time `json:"time"`
	TxCount int       `json:"transaction_count"`
	TxIDs   []bc.Hash `json:"transaction_ids"`
}

// GetBlockSummary returns header data for the requested block.
func GetBlockSummary(ctx context.Context, hash string) (*BlockSummary, error) {
	block, err := txdb.GetBlock(ctx, hash)
	if err != nil {
		return nil, err
	}

	txHashes := make([]bc.Hash, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.Hash)
	}

	return &BlockSummary{
		ID:      block.Hash(),
		Height:  block.Height,
		Time:    block.Time(),
		TxCount: len(block.Transactions),
		TxIDs:   txHashes,
	}, nil
}

// Tx is returned by GetTx
type Tx struct {
	ID          bc.Hash            `json:"id"`
	BlockID     *bc.Hash           `json:"block_id"`
	BlockHeight uint64             `json:"block_height"`
	BlockTime   time.Time          `json:"block_time"`
	Inputs      []*TxInput         `json:"inputs"`
	Outputs     []*TxOutput        `json:"outputs"`
	Metadata    chainjson.HexBytes `json:"metadata,omitempty"`
}

// TxInput is an input in a Tx
type TxInput struct {
	Type     string             `json:"type"`
	TxID     *bc.Hash           `json:"transaction_id,omitempty"`
	TxOut    *uint32            `json:"transaction_output,omitempty"`
	AssetID  bc.AssetID         `json:"asset_id"`
	Amount   *uint64            `json:"amount,omitempty"`
	Metadata chainjson.HexBytes `json:"metadata,omitempty"`
	AssetDef chainjson.HexBytes `json:"asset_definition,omitempty"`
}

// TxOutput is an output in a Tx
type TxOutput struct {
	AssetID  bc.AssetID         `json:"asset_id"`
	Amount   uint64             `json:"amount"`
	Address  chainjson.HexBytes `json:"address"` // deprecated
	Script   chainjson.HexBytes `json:"script"`
	Metadata chainjson.HexBytes `json:"metadata,omitempty"`
}

// GetTx returns a transaction with additional details added.
func GetTx(ctx context.Context, txID string) (*Tx, error) {
	hash, err := bc.ParseHash(txID)
	if err != nil {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}

	txs, err := txdb.GetTxs(ctx, hash)
	if err != nil {
		return nil, err
	}
	tx, ok := txs[hash]
	if !ok {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}

	blockHeader, err := txdb.GetTxBlockHeader(ctx, txID)
	if err != nil {
		return nil, err
	}

	var inHashes []bc.Hash
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		inHashes = append(inHashes, in.Previous.Hash)
	}
	prevTxs, err := txdb.GetTxs(ctx, inHashes...)

	if err != nil {
		return nil, errors.Wrap(err, "fetching inputs")
	}

	return makeTx(tx, blockHeader, prevTxs)
}

// Asset is returned by GetAsset
type Asset struct {
	ID            bc.AssetID         `json:"id"`
	DefinitionPtr string             `json:"definition_pointer"`
	Definition    chainjson.HexBytes `json:"definition"`
	Issued        uint64             `json:"issued"`
}

// GetAssets returns data about the specified assets, including the most recent
// asset definition submitted for each asset. If a given asset ID is not found,
// that asset will not be included in the response.
func GetAssets(ctx context.Context, assetIDs []string) (map[string]*Asset, error) {
	// TODO(jeffomatic): This function makes use of the assets and
	// issuance_totals tables, which technically violates the line between
	// issuer nodes and explorer nodes.
	//
	// We do this because we require:
	// 1. issued totals, which are only tracked in the issuance_totals table.
	// 2. assets with blank asset defs, which appear in the assets table, but
	//    not the asset_definition_pointers table. This is a bug.
	//
	// As a result, the explorer node will return entries for assets that have
	// not yet been issued, or whose issuances have not yet landed in a block.
	// For these results, the asset definition will appear to be blank.

	res := make(map[string]*Asset)

	withCirc, err := appdb.GetAssets(ctx, assetIDs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch issuer node asset data")
	}

	for id, inodeAsset := range withCirc {
		res[id] = &Asset{
			ID:     inodeAsset.ID,
			Issued: inodeAsset.Issued.Confirmed,
		}
	}

	defs, err := txdb.AssetDefinitions(ctx, assetIDs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch txdb asset def")
	}

	for id, def := range defs {
		p := bc.HashAssetDefinition(def).String()
		d := chainjson.HexBytes(def)

		if a, ok := res[id]; ok {
			a.DefinitionPtr = p
			a.Definition = d
		} else {
			// Ignore missing asset defs. It could mean the asset hasn't
			// landed yet, or it could mean that the asset def was blank.

			aid := new(bc.AssetID)
			err := aid.UnmarshalText([]byte(id))
			if err != nil {
				// should never happen
				return nil, errors.Wrap(err, "invalid asset id:", id)
			}

			res[id] = &Asset{
				ID:            *aid,
				DefinitionPtr: p,
				Definition:    d,
			}
		}
	}

	return res, nil
}

// GetAsset returns the most recent asset definition stored in
// the blockchain, for the given asset.
func GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	assets, err := GetAssets(ctx, []string{assetID})
	if err != nil {
		return nil, err
	}

	a, ok := assets[assetID]
	if !ok {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "asset ID: %q", assetID)
	}

	return a, nil
}

func makeTx(bcTx *bc.Tx, blockHeader *bc.BlockHeader, prevTxs map[bc.Hash]*bc.Tx) (*Tx, error) {
	resp := &Tx{
		ID:       bcTx.Hash,
		Metadata: bcTx.Metadata,
	}

	bhash := blockHeader.Hash()
	resp.BlockID = &bhash
	resp.BlockHeight = blockHeader.Height
	resp.BlockTime = blockHeader.Time()

	for _, in := range bcTx.Inputs {
		if in.IsIssuance() {
			redeemScript, err := txscript.RedeemScriptFromP2SHSigScript(in.SignatureScript)
			if err != nil {
				return nil, errors.Wrap(err, "extracting redeem script from sigscript")
			}
			pkScript := txscript.RedeemToPkScript(redeemScript)
			assetID := bc.ComputeAssetID(pkScript, [32]byte{}) // TODO(tessr): get genesis hash

			resp.Inputs = append(resp.Inputs, &TxInput{
				Type:     "issuance",
				AssetID:  assetID,
				Metadata: in.Metadata,
				AssetDef: in.AssetDefinition,
			})
		} else {
			prevTx, ok := prevTxs[in.Previous.Hash]
			if !ok {
				return nil, errors.Wrap(fmt.Errorf("missing previous transaction %s", in.Previous.Hash))
			}

			if in.Previous.Index >= uint32(len(prevTx.Outputs)) {
				return nil, errors.Wrap(fmt.Errorf("transaction %s missing output %d", in.Previous.Hash, in.Previous.Index))
			}

			resp.Inputs = append(resp.Inputs, &TxInput{
				Type:     "transfer",
				AssetID:  prevTx.Outputs[in.Previous.Index].AssetID,
				Amount:   &prevTx.Outputs[in.Previous.Index].Amount,
				TxID:     &in.Previous.Hash,
				TxOut:    &in.Previous.Index,
				Metadata: in.Metadata,
			})
		}

	}

	for _, out := range bcTx.Outputs {
		resp.Outputs = append(resp.Outputs, &TxOutput{
			AssetID:  out.AssetID,
			Amount:   out.Amount,
			Address:  out.Script,
			Script:   out.Script,
			Metadata: out.Metadata,
		})
	}

	return resp, nil
}
