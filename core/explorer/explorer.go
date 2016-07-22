package explorer

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/core/txdb"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
)

// Explorer records blockchain history
// and provides functions to look up
// blocks, transactions, and other records.
type Explorer struct {
	db         pg.DB
	store      *txdb.Store // TODO(kr): get rid of this
	maxAge     time.Duration
	historical bool
	isManager  bool

	lastPrune time.Time
}

// New makes a new Explorer storing its state in db,
// with a block callback in fc for indexing utxos in the
// explorer_outputs table and occasionally pruning ones spent
// spent more than maxAgeDays ago.  (If maxAgeDays is <= 0, no
// pruning is done.)
func New(fc *cos.FC, db pg.DB, store *txdb.Store, maxAge time.Duration, historical, isManager bool) *Explorer {
	e := &Explorer{
		db:         db,
		store:      store,
		historical: historical,
		maxAge:     maxAge,
		isManager:  isManager,
	}
	fc.AddBlockCallback(e.addBlock)
	return e
}

func (e *Explorer) addBlock(ctx context.Context, block *bc.Block) {
	e.indexHistoricalBlock(ctx, block)
}

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
func (e *Explorer) ListBlocks(ctx context.Context, prev string, limit int) ([]ListBlocksItem, string, error) {
	blocks, err := e.store.ListBlocks(ctx, prev, limit)
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
func (e *Explorer) GetBlockSummary(ctx context.Context, hash string) (*BlockSummary, error) {
	block, err := e.store.GetBlockByHash(ctx, hash)
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
// and not rely on the Store.
func (e *Explorer) GetTx(ctx context.Context, txHashStr string) (*Tx, error) {
	hash, err := bc.ParseHash(txHashStr)
	if err != nil {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}

	txs, err := e.store.GetTxs(ctx, hash)
	if err != nil {
		return nil, err
	}
	tx, ok := txs[hash]
	if !ok {
		return nil, errors.Wrap(pg.ErrUserInputNotFound)
	}

	blockHeader, err := e.store.GetTxBlockHeader(ctx, hash)
	if err != nil {
		return nil, err
	}

	var inHashes []bc.Hash
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		inHashes = append(inHashes, in.Outpoint().Hash)
	}

	prevTxs, err := e.store.GetTxs(ctx, inHashes...)
	if err != nil {
		return nil, errors.Wrap(err, "fetching bc inputs")
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
func (e *Explorer) GetAssets(ctx context.Context, assetIDs []bc.AssetID) (map[bc.AssetID]*Asset, error) {
	circ, err := asset.Circulation(pg.NewContext(ctx, e.db), assetIDs...)
	if err != nil {
		return nil, errors.Wrap(err, "fetch asset circulation data")
	}

	defs, err := asset.Definitions(pg.NewContext(ctx, e.db), assetIDs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch txdb asset def")
	}

	res := make(map[bc.AssetID]*Asset)
	for assetID, amount := range circ.Assets {
		res[assetID] = &Asset{
			ID:     assetID,
			Issued: amount.Issued,
		}
	}
	for id, def := range defs {
		p := bc.HashAssetDefinition(def).String()
		d := chainjson.HexBytes(def)

		if a, ok := res[id]; ok {
			a.DefinitionPtr = p
			a.Definition = d
		} else {
			// Ignore missing asset defs.
			res[id] = &Asset{
				ID:            id,
				DefinitionPtr: p,
				Definition:    d,
			}
		}
	}
	return res, nil
}

// GetAsset returns the most recent asset definition stored in
// the blockchain, for the given asset.
func (e *Explorer) GetAsset(ctx context.Context, assetID bc.AssetID) (*Asset, error) {
	assets, err := e.GetAssets(ctx, []bc.AssetID{assetID})
	if err != nil {
		return nil, err
	}
	a, ok := assets[assetID]
	if !ok {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "asset ID: %q", assetID.String())
	}
	return a, nil
}

func (e *Explorer) ListUTXOsByAsset(ctx context.Context, assetID bc.AssetID, prev string, limit int) ([]*TxOutput, string, error) {
	return e.listHistoricalOutputsByAssetAndAccount(ctx, assetID, "", time.Now(), prev, limit)
}

func stateOutsToTxOuts(stateOuts []*state.Output) []*TxOutput {
	var res []*TxOutput
	for _, sOut := range stateOuts {
		res = append(res, &TxOutput{
			TxHash:   &sOut.Outpoint.Hash,
			TxIndex:  &sOut.Outpoint.Index,
			AssetID:  sOut.AssetID,
			Amount:   sOut.Amount,
			Address:  sOut.ControlProgram,
			Script:   sOut.ControlProgram,
			Metadata: sOut.ReferenceData,
		})
	}

	return res
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
			resp.Inputs = append(resp.Inputs, &TxInput{
				Type:     "issuance",
				AssetID:  in.AssetID(),
				Metadata: in.ReferenceData,
				AssetDef: in.AssetDefinition(),
			})
		} else {
			o := in.Outpoint()
			prevTx, ok := prevTxs[o.Hash]
			if !ok {
				return nil, errors.Wrap(fmt.Errorf("missing previous transaction %s", o.Hash))
			}

			if o.Index >= uint32(len(prevTx.Outputs)) {
				return nil, errors.Wrap(fmt.Errorf("transaction %s missing output %d", o.Hash, o.Index))
			}

			resp.Inputs = append(resp.Inputs, &TxInput{
				Type:     "transfer",
				AssetID:  prevTx.Outputs[o.Index].AssetID,
				Amount:   &prevTx.Outputs[o.Index].Amount,
				TxHash:   &o.Hash,
				TxOut:    &o.Index,
				Metadata: in.ReferenceData,
			})
		}

	}

	for _, out := range bcTx.Outputs {
		resp.Outputs = append(resp.Outputs, &TxOutput{
			AssetID:  out.AssetID,
			Amount:   out.Amount,
			Address:  out.ControlProgram,
			Script:   out.ControlProgram,
			Metadata: out.ReferenceData,
		})
	}

	return resp, nil
}
