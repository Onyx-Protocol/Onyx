package asset

import (
	"database/sql"
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/core/signers"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
)

var (
	ErrArchived = errors.New("asset archived")
)

type Asset struct {
	AssetID         bc.AssetID      `json:"id"`
	Definition      []byte          `json:"definition"`
	IssuanceProgram []byte          `json:"issuance_program"`
	RedeemProgram   []byte          `json:"redeem_program"`
	GenesisHash     bc.Hash         `json:"genesis_hash"`
	Signer          *signers.Signer `json:"signer"`
	KeyIndex        []uint32        `json:"key_index"`
}

// Define defines a new Asset.
func Define(ctx context.Context, xpubs []string, quorum int, definition map[string]interface{}, genesisHash bc.Hash, clientToken *string) (*Asset, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "define asset")
	}
	defer dbtx.Rollback(ctx)
	assetSigner, err := signers.Create(ctx, "asset", xpubs, quorum, clientToken)
	if err != nil {
		return nil, err
	}

	def, err := serializeAssetDef(definition)
	if err != nil {
		return nil, errors.Wrap(err, "serializing asset definition")
	}

	idx, err := nextIndex(ctx)
	if err != nil {
		return nil, err
	}

	path := signers.Path(assetSigner, signers.AssetKeySpace, idx)

	derivedXPubs := hd25519.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := hd25519.XPubKeys(derivedXPubs)
	issuanceProgram, redeem, err := txscript.Scripts(derivedPKs, assetSigner.Quorum)
	if err != nil {
		return nil, err
	}

	asset := &Asset{
		KeyIndex:        idx,
		Definition:      def,
		IssuanceProgram: issuanceProgram,
		RedeemProgram:   redeem,
		GenesisHash:     genesisHash,
		AssetID:         bc.ComputeAssetID(issuanceProgram, genesisHash, 1),
		Signer:          assetSigner,
	}

	asset, err = insertAsset(ctx, asset, clientToken)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing define asset dbtx")
	}
	return asset, nil
}

// Find retrieves an Asset record from signer.
func Find(ctx context.Context, id bc.AssetID) (*Asset, error) {
	asset, err := assetByAssetID(ctx, id)
	if err != nil {
		return nil, err
	}

	asset.Signer, err = signers.Find(ctx, "asset", asset.Signer.ID)
	if err != nil {
		return nil, err
	}
	return asset, nil
}

// Archive marks an Asset record as archived, effectively "deleting" it.
func Archive(ctx context.Context, id bc.AssetID) error {
	asset, err := assetByAssetID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "asset is missing")
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "archive asset")
	}
	defer dbtx.Rollback(ctx)

	const q = `UPDATE assets SET archived = true WHERE id = $1`
	_, err = pg.Exec(ctx, q, id.String())
	if err != nil {
		return errors.Wrap(err, "archive asset query")
	}
	err = signers.Archive(ctx, "asset", asset.Signer.ID)
	if err != nil {
		return errors.Wrap(err, "archive asset signer")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return errors.Wrap(err, "committing archive asset dbtx")
	}

	return nil
}

// List returns a paginated set of Assets
func List(ctx context.Context, prev string, limit int) ([]*Asset, string, error) {
	assetSigners, last, err := signers.List(ctx, "asset", prev, limit)
	if err != nil {
		return nil, "", err
	}
	// TODO(tessr): fetch asset definition

	assets := make([]*Asset, 0, len(assetSigners))
	for _, signer := range assetSigners {
		a := &Asset{
			Signer: signer,
		}
		assets = append(assets, a)
	}

	return assets, last, nil
}

// insertAsset adds the asset to the database. If the asset has a client token,
// and there already exists an asset for the same issuer node with that client
// token, insertAsset will lookup and return the existing asset instead.
func insertAsset(ctx context.Context, asset *Asset, clientToken *string) (*Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
    INSERT INTO assets
	 	(id, signer_id, key_index, redeem_program, genesis_hash, issuance_program, definition, client_token)
    VALUES($1, $2, to_key_index($3), $4, $5, $6, $7, $8)
    ON CONFLICT (client_token) DO NOTHING
  `

	res, err := pg.Exec(
		ctx, q,
		asset.AssetID, asset.Signer.ID, pg.Uint32s(asset.KeyIndex),
		asset.RedeemProgram, asset.GenesisHash, asset.IssuanceProgram,
		asset.Definition, clientToken,
	)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving rows affected")
	}
	if rowsAffected == 0 && clientToken != nil {
		// There is already an asset for this issuer node with the provided client
		// token. We should return the existing asset.
		asset, err = assetByClientToken(ctx, *clientToken)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving existing asset")
		}
	}
	return asset, nil
}

func assetByAssetID(ctx context.Context, id bc.AssetID) (*Asset, error) {
	// TODO: fetch asset definition as well
	const q = `
		SELECT id, issuance_program, redeem_program,
			genesis_hash, key_index(key_index), signer_id, archived
		FROM assets
		WHERE id=$1
	`

	var (
		a        Asset
		archived bool
		signerID string
	)

	err := pg.QueryRow(ctx, q, id.String()).Scan(
		&a.AssetID,
		&a.IssuanceProgram,
		&a.RedeemProgram,
		&a.GenesisHash,
		(*pg.Uint32s)(&a.KeyIndex),
		&signerID,
		&archived,
	)

	if err != nil {
		return nil, err
	}

	if archived {
		return nil, ErrArchived
	}

	sig, err := signers.Find(ctx, "asset", signerID)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't find signer")
	}

	a.Signer = sig
	return &a, nil
}

// assetByClientToken loads an asset from the database using its client token.
func assetByClientToken(ctx context.Context, clientToken string) (*Asset, error) {
	// TODO: fetch asset definition as well
	const q = `
		SELECT id, redeem_program, issuance_program, genesis_hash, key_index(key_index),
			signer_id, archived
		FROM assets
		WHERE client_token=$1
	`
	var (
		a        Asset
		archived bool
		signerID string
	)
	err := pg.QueryRow(ctx, q, clientToken).Scan(
		&a.AssetID,
		&a.RedeemProgram,
		&a.IssuanceProgram,
		&a.GenesisHash,
		(*pg.Uint32s)(&a.KeyIndex),
		&signerID,
		&archived,
	)
	if err != nil {
		return nil, err
	}

	if archived {
		return nil, ErrArchived
	}

	sig, err := signers.Find(ctx, "asset", signerID)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't find signer")
	}

	a.Signer = sig
	return &a, nil
}

// serializeAssetDef produces a canonical byte representation of an asset
// definition. Currently, this is implemented using pretty-printed JSON.
// As is the standard for Go's map[string] serialization, object keys will
// appear in lexicographic order. Although this is mostly meant for machine
// consumption, the JSON is pretty-printed for easy reading.
func serializeAssetDef(def map[string]interface{}) ([]byte, error) {
	if def == nil {
		return nil, nil
	}
	return json.MarshalIndent(def, "", "  ")
}

func nextIndex(ctx context.Context) ([]uint32, error) {
	const q = `SELECT key_index(nextval('assets_key_index_seq'::regclass)-1)`
	var idx []uint32
	err := pg.QueryRow(ctx, q).Scan((*pg.Uint32s)(&idx))
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, errors.WithDetailf(err, "get key info")
	}
	return idx, nil
}
