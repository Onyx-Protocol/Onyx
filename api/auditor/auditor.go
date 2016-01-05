package auditor

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain/bc"
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
	txs, err := txdb.GetTxs(ctx, txID)
	if err != nil {
		return nil, err
	}
	tx, ok := txs[txID]
	if !ok {
		return nil, errors.New("tx not found: " + txID)
	}

	resp := &Tx{
		ID:       tx.Hash,
		Metadata: tx.Metadata,
	}

	blockHeader, err := txdb.GetTxBlockHeader(ctx, txID)
	if err != nil {
		return nil, err
	}

	if blockHeader != nil {
		bhash := blockHeader.Hash()
		resp.BlockID = &bhash
		resp.BlockHeight = blockHeader.Height
		resp.BlockTime = blockHeader.Time()
	}

	if tx.IsIssuance() {
		if len(tx.Outputs) == 0 {
			return nil, errors.New("invalid transaction")
		}

		resp.Inputs = append(resp.Inputs, &TxInput{
			Type:     "issuance",
			AssetID:  tx.Outputs[0].AssetID,
			Metadata: tx.Inputs[0].Metadata,
			AssetDef: tx.Inputs[0].AssetDefinition,
		})
	} else {
		var inHashes []string
		for _, in := range tx.Inputs {
			inHashes = append(inHashes, in.Previous.Hash.String())
		}
		txs, err = txdb.GetTxs(ctx, inHashes...)
		if err != nil {
			if errors.Root(err) == pg.ErrUserInputNotFound {
				err = sql.ErrNoRows
			}
			return nil, errors.Wrap(err, "fetching inputs")
		}
		for _, in := range tx.Inputs {
			prev := txs[in.Previous.Hash.String()].Outputs[in.Previous.Index]
			resp.Inputs = append(resp.Inputs, &TxInput{
				Type:     "transfer",
				AssetID:  prev.AssetID,
				Amount:   &prev.Value,
				TxID:     &in.Previous.Hash,
				TxOut:    &in.Previous.Index,
				Metadata: in.Metadata,
			})
		}
	}

	for _, out := range tx.Outputs {
		resp.Outputs = append(resp.Outputs, &TxOutput{
			AssetID:  out.AssetID,
			Amount:   out.Value,
			Address:  out.Script,
			Script:   out.Script,
			Metadata: out.Metadata,
		})
	}

	return resp, nil
}

// Asset is returned by GetAsset
type Asset struct {
	ID            bc.AssetID         `json:"id"`
	DefinitionPtr string             `json:"definition_pointer"`
	Definition    chainjson.HexBytes `json:"definition"`
	Issued        uint64             `json:"issued"`
}

// GetAsset returns the most recent asset definition stored in
// the blockchain, for the given asset.
func GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	// TODO(erykwalder): replace with a txdb call
	asset, err := appdb.GetAsset(ctx, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "loading asset")
	}

	// Ignore missing asset defs
	hash, def, err := txdb.AssetDefinition(ctx, assetID)
	if err != nil && errors.Root(err) != pg.ErrUserInputNotFound {
		return nil, errors.Wrap(err, "loading definition")
	}

	return &Asset{asset.ID, hash, def, asset.Issued.Confirmed}, nil
}
