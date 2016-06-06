package explorer

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txdb"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
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
func ListBlocks(ctx context.Context, store *txdb.Store, prev string, limit int) ([]ListBlocksItem, string, error) {
	blocks, err := store.ListBlocks(ctx, prev, limit)
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
	ID       bc.Hash   `json:"id"`
	Height   uint64    `json:"height"`
	Time     time.Time `json:"time"`
	TxCount  int       `json:"transaction_count"`
	TxHashes []bc.Hash `json:"transaction_ids"`
}

// GetBlockSummary returns header data for the requested block.
func GetBlockSummary(ctx context.Context, store *txdb.Store, hash string) (*BlockSummary, error) {
	block, err := store.GetBlock(ctx, hash)
	if err != nil {
		return nil, err
	}

	txHashes := make([]bc.Hash, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.Hash)
	}

	return &BlockSummary{
		ID:       block.Hash(),
		Height:   block.Height,
		Time:     block.Time(),
		TxCount:  len(block.Transactions),
		TxHashes: txHashes,
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
	TxHash   *bc.Hash           `json:"transaction_id,omitempty"`
	TxOut    *uint32            `json:"transaction_output,omitempty"`
	AssetID  bc.AssetID         `json:"asset_id"`
	Amount   *uint64            `json:"amount,omitempty"`
	Metadata chainjson.HexBytes `json:"metadata,omitempty"`
	AssetDef chainjson.HexBytes `json:"asset_definition,omitempty"`
}

// TxOutput is an output in a Tx
type TxOutput struct {
	// TxHash and TxIndex should only be populated if you're returning TxOutputs
	// directly outside of a Tx
	TxHash  *bc.Hash `json:"transaction_id,omitempty"`
	TxIndex *uint32  `json:"transaction_output,omitempty"`

	AssetID  bc.AssetID         `json:"asset_id"`
	Amount   uint64             `json:"amount"`
	Address  chainjson.HexBytes `json:"address"` // deprecated
	Script   chainjson.HexBytes `json:"script"`
	Metadata chainjson.HexBytes `json:"metadata,omitempty"`
}

// GetTx returns a transaction with additional details added.
// TODO(jackson): Explorer should do its own indexing of transactions
// and not rely on the Store or Pool.
func GetTx(ctx context.Context, store *txdb.Store, pool *txdb.Pool, txHashStr string) (*Tx, error) {
	hash, err := bc.ParseHash(txHashStr)
	if err != nil {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}

	poolTxs, err := pool.GetTxs(ctx, hash)
	if err != nil {
		return nil, err
	}
	bcTxs, err := store.GetTxs(ctx, hash)
	if err != nil {
		return nil, err
	}
	tx, ok := poolTxs[hash]
	if !ok {
		tx, ok = bcTxs[hash]
	}
	if !ok {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}

	blockHeader, err := store.GetTxBlockHeader(ctx, hash)
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

	prevPoolTxs, err := pool.GetTxs(ctx, inHashes...)
	if err != nil {
		return nil, errors.Wrap(err, "fetching pool inputs")
	}
	prevBcTxs, err := store.GetTxs(ctx, inHashes...)
	if err != nil {
		return nil, errors.Wrap(err, "fetching bc inputs")
	}

	return makeTx(tx, blockHeader, prevPoolTxs, prevBcTxs)
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
func GetAssets(ctx context.Context, store *txdb.Store, assetIDs []string) (map[string]*Asset, error) {
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
		return nil, errors.Wrap(err, "fetch asset issuer asset data")
	}

	for id, inodeAsset := range withCirc {
		res[id] = &Asset{
			ID:     inodeAsset.ID,
			Issued: inodeAsset.Issued.Confirmed,
		}
	}

	defs, err := store.AssetDefinitions(ctx, assetIDs)
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
func GetAsset(ctx context.Context, store *txdb.Store, assetID string) (*Asset, error) {
	assets, err := GetAssets(ctx, store, []string{assetID})
	if err != nil {
		return nil, err
	}

	a, ok := assets[assetID]
	if !ok {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "asset ID: %q", assetID)
	}

	return a, nil
}

func ListUTXOsByAsset(ctx context.Context, assetID bc.AssetID, prev string, limit int) ([]*TxOutput, string, error) {
	return listHistoricalOutputsByAssetAndAccount(ctx, assetID, "", time.Now(), prev, limit)
}

func stateOutsToTxOuts(stateOuts []*state.Output) []*TxOutput {
	var res []*TxOutput
	for _, sOut := range stateOuts {
		res = append(res, &TxOutput{
			TxHash:   &sOut.Outpoint.Hash,
			TxIndex:  &sOut.Outpoint.Index,
			AssetID:  sOut.AssetID,
			Amount:   sOut.Amount,
			Address:  sOut.Script,
			Script:   sOut.Script,
			Metadata: sOut.Metadata,
		})
	}

	return res
}

func makeTx(bcTx *bc.Tx, blockHeader *bc.BlockHeader, prevPoolTxs, prevBcTxs map[bc.Hash]*bc.Tx) (*Tx, error) {
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
			prevTx, ok := prevPoolTxs[in.Previous.Hash]
			if !ok {
				prevTx, ok = prevBcTxs[in.Previous.Hash]
				if !ok {
					return nil, errors.Wrap(fmt.Errorf("missing previous transaction %s", in.Previous.Hash))
				}
			}

			if in.Previous.Index >= uint32(len(prevTx.Outputs)) {
				return nil, errors.Wrap(fmt.Errorf("transaction %s missing output %d", in.Previous.Hash, in.Previous.Index))
			}

			resp.Inputs = append(resp.Inputs, &TxInput{
				Type:     "transfer",
				AssetID:  prevTx.Outputs[in.Previous.Index].AssetID,
				Amount:   &prevTx.Outputs[in.Previous.Index].Amount,
				TxHash:   &in.Previous.Hash,
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
