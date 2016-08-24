package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"chain/core/signers"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

var (
	ErrArchived = errors.New("asset archived")
)

type Asset struct {
	AssetID         bc.AssetID             `json:"id"`
	Alias           string                 `json:"alias"`
	Definition      map[string]interface{} `json:"definition"`
	IssuanceProgram []byte                 `json:"issuance_program"`
	GenesisHash     bc.Hash                `json:"genesis_hash"`
	Signer          *signers.Signer        `json:"signer"`
	Tags            map[string]interface{} `json:"tags"`
}

// Define defines a new Asset.
func Define(ctx context.Context, xpubs []string, quorum int, definition map[string]interface{}, genesisHash bc.Hash, alias string, tags map[string]interface{}, clientToken *string) (*Asset, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "define asset")
	}
	defer dbtx.Rollback(ctx)
	assetSigner, err := signers.Create(ctx, "asset", xpubs, quorum, clientToken)
	if err != nil {
		return nil, err
	}

	serializedDef, err := serializeAssetDef(definition)
	if err != nil {
		return nil, errors.Wrap(err, "serializing asset definition")
	}

	path := signers.Path(assetSigner, signers.AssetKeySpace, nil)

	derivedXPubs := hd25519.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := hd25519.XPubKeys(derivedXPubs)
	issuanceProgram, err := programWithDefinition(derivedPKs, assetSigner.Quorum, serializedDef)
	if err != nil {
		return nil, err
	}

	asset := &Asset{
		Alias:           alias,
		Definition:      definition,
		IssuanceProgram: issuanceProgram,
		GenesisHash:     genesisHash,
		AssetID:         bc.ComputeAssetID(issuanceProgram, genesisHash, 1),
		Signer:          assetSigner,
		Tags:            tags,
	}

	asset, err = insertAsset(ctx, asset, clientToken)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset")
	}

	err = insertAssetTags(ctx, asset.AssetID, tags)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset tags")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing define asset dbtx")
	}

	// Note, this should be okay to do outside of the SQL txn
	// because each step should be idempotent. Also, we have no
	// guarantee that the query engine uses the same db handle.
	err = indexAnnotatedAsset(ctx, asset)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated asset")
	}

	return asset, nil
}

// SetTags sets tags on the given asset and its associated signer.
func SetTags(ctx context.Context, id bc.AssetID, alias string, newTags map[string]interface{}) (*Asset, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "set asset tags")
	}
	defer dbtx.Rollback(ctx)

	var a *Asset
	if (id != bc.AssetID{}) {
		err = insertAssetTags(ctx, id, newTags)
		if err != nil {
			return nil, errors.Wrap(err, "set asset tags")
		}

		a, err = assetByAssetID(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	} else {
		a, err = assetByAlias(ctx, alias)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = insertAssetTags(ctx, a.AssetID, newTags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		a.Tags = newTags
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing set asset tags dbtx")
	}

	// Note, this should be okay to do outside of the SQL txn
	// because each step should be idempotent. Also, we have no
	// guarantee that the query engine uses the same db handle.
	err = indexAnnotatedAsset(ctx, a)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated asset")
	}

	return a, nil
}

// FindByID retrieves an Asset record along with its signer, given an assetID.
func FindByID(ctx context.Context, id bc.AssetID) (*Asset, error) {
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

// FindByAlias retrieves an Asset record along with its signer,
// given an asset alias.
func FindByAlias(ctx context.Context, alias string) (*Asset, error) {
	asset, err := assetByAlias(ctx, alias)
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
func Archive(ctx context.Context, id bc.AssetID, alias string) error {
	var (
		asset *Asset
		err   error
	)
	if (id != bc.AssetID{}) {
		asset, err = assetByAssetID(ctx, id)
		if err != nil {
			return errors.Wrap(err, "asset is missing")
		}
	} else {
		asset, err = assetByAlias(ctx, alias)
		if err != nil {
			return errors.Wrap(err, "asset is missing")
		}
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "archive asset")
	}
	defer dbtx.Rollback(ctx)

	const q = `UPDATE assets SET archived = true WHERE id = $1`
	_, err = pg.Exec(ctx, q, asset.AssetID.String())
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

// FindBatch returns a map of Assets for the provided IDs. The
// asset tags on the returned Assets will not be populated.
func FindBatch(ctx context.Context, assetIDs ...bc.AssetID) (map[string]*Asset, error) {
	const q = `
		SELECT assets.id, definition, issuance_program, signer_id,
			quorum, xpubs, key_index(signers.key_index)
		FROM assets
		LEFT JOIN signers ON (assets.signer_id=signers.id)
		WHERE assets.id = ANY($1) AND NOT assets.archived AND signers.type='asset'
	`

	assetIDStrings := make([]string, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		assetIDStrings = append(assetIDStrings, assetID.String())
	}

	assets := make(map[string]*Asset, len(assetIDs))
	err := pg.ForQueryRows(ctx, q, pg.Strings(assetIDStrings),
		func(id string, definitionBytes []byte, issuanceProgram []byte, signerID string, quorum int, xpubs pg.Strings, keyIndex pg.Uint32s) error {
			var assetID bc.AssetID
			err := assetID.UnmarshalText([]byte(id))
			if err != nil {
				return errors.WithDetailf(httpjson.ErrBadRequest, "%q is an invalid asset ID", assetID)
			}

			keys, err := signers.ConvertKeys(xpubs)
			if err != nil {
				return errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
			}

			var definition map[string]interface{}
			if len(definitionBytes) > 0 {
				err := json.Unmarshal(definitionBytes, &definition)
				if err != nil {
					return errors.Wrap(err)
				}
			}

			assets[id] = &Asset{
				AssetID:         assetID,
				Definition:      definition,
				IssuanceProgram: issuanceProgram,
				Signer: &signers.Signer{
					ID:       signerID,
					Type:     "asset",
					XPubs:    keys,
					Quorum:   quorum,
					KeyIndex: keyIndex,
				},
			}
			return nil
		})
	return assets, errors.Wrap(err)
}

// insertAsset adds the asset to the database. If the asset has a client token,
// and there already exists an asset for the same issuer node with that client
// token, insertAsset will lookup and return the existing asset instead.
func insertAsset(ctx context.Context, asset *Asset, clientToken *string) (*Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
    INSERT INTO assets
	 	(id, alias, signer_id, genesis_hash, issuance_program, definition, client_token)
    VALUES($1, $2, $3, $4, $5, $6, $7)
    ON CONFLICT (client_token) DO NOTHING
  `
	defParams, err := mapToNullString(asset.Definition)
	if err != nil {
		return nil, err
	}

	aliasParam := sql.NullString{
		String: asset.Alias,
		Valid:  asset.Alias != "",
	}

	res, err := pg.Exec(
		ctx, q,
		asset.AssetID, aliasParam, asset.Signer.ID,
		asset.GenesisHash, asset.IssuanceProgram,
		defParams, clientToken,
	)
	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "non-unique alias")
	} else if err != nil {
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

// insertAssetTags inserts a set of tags for the given assetID.
// It must take place inside a database transaction.
func insertAssetTags(ctx context.Context, assetID bc.AssetID, tags map[string]interface{}) error {
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	tagsParam, err := mapToNullString(tags)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `
		INSERT INTO asset_tags (asset_id, tags) VALUES ($1, $2)
		ON CONFLICT (asset_id) DO UPDATE SET tags = $2
	`
	_, err = pg.Exec(ctx, q, assetID.String(), tagsParam)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func assetByAssetID(ctx context.Context, id bc.AssetID) (*Asset, error) {
	const q = `
		SELECT id, alias, issuance_program, definition, genesis_hash, signer_id, archived
		FROM assets
		WHERE id=$1
	`

	var (
		a          Asset
		alias      sql.NullString
		archived   bool
		signerID   string
		definition []byte
	)

	err := pg.QueryRow(ctx, q, id.String()).Scan(
		&a.AssetID,
		&alias,
		&a.IssuanceProgram,
		&definition,
		&a.GenesisHash,
		&signerID,
		&archived,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if archived {
		return nil, ErrArchived
	}

	if err == sql.ErrNoRows {
		// Assume that this is a non-local asset
		// if we can't find it in the assets table
		a = Asset{AssetID: id}
	} else {
		// Only try to fetch the signer if this is a
		// local asset.
		sig, err := signers.Find(ctx, "asset", signerID)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't find signer")
		}

		a.Signer = sig
	}

	if len(definition) > 0 {
		err := json.Unmarshal(definition, &a.Definition)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	if alias.Valid {
		a.Alias = alias.String
	}

	const tagQ = `SELECT tags FROM asset_tags WHERE asset_id=$1`
	var tags []byte
	err = pg.QueryRow(ctx, tagQ, id.String()).Scan(&tags)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if len(tags) > 0 {
		err := json.Unmarshal(tags, &a.Tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	return &a, nil
}

func assetByAlias(ctx context.Context, alias string) (*Asset, error) {
	const q = `
		SELECT id, alias, issuance_program, definition, genesis_hash, signer_id, archived
		FROM assets
		WHERE alias=$1
	`

	var (
		a          Asset
		archived   bool
		signerID   string
		definition []byte
	)

	err := pg.QueryRow(ctx, q, alias).Scan(
		&a.AssetID,
		&a.Alias,
		&a.IssuanceProgram,
		&definition,
		&a.GenesisHash,
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

	if len(definition) > 0 {
		err := json.Unmarshal(definition, &a.Definition)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	const tagQ = `SELECT tags FROM asset_tags WHERE asset_id=$1`
	var tags []byte
	err = pg.QueryRow(ctx, tagQ, a.AssetID).Scan(&tags)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if len(tags) > 0 {
		err := json.Unmarshal(tags, &a.Tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	return &a, nil
}

// assetByClientToken loads an asset from the database using its client token.
func assetByClientToken(ctx context.Context, clientToken string) (*Asset, error) {
	const q = `
		SELECT id, issuance_program, definition,
			genesis_hash, signer_id, archived
		FROM assets
		WHERE client_token=$1
	`
	var (
		a          Asset
		archived   bool
		signerID   string
		definition []byte
	)
	err := pg.QueryRow(ctx, q, clientToken).Scan(
		&a.AssetID,
		&a.IssuanceProgram,
		&definition,
		&a.GenesisHash,
		&signerID,
		&archived,
	)
	if err != nil {
		return nil, err
	}

	if archived {
		return nil, ErrArchived
	}

	if len(definition) > 0 {
		err := json.Unmarshal(definition, &a.Definition)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	const tagQ = `SELECT tags FROM asset_tags WHERE asset_id=$1`
	var tags []byte
	err = pg.QueryRow(ctx, tagQ, a.AssetID.String()).Scan(&tags)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if len(tags) > 0 {
		err := json.Unmarshal(tags, &a.Tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
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
// The empty asset def is an empty byte slice.
func serializeAssetDef(def map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(def, "", "  ")
}

func programWithDefinition(pubkeys []ed25519.PublicKey, nrequired int, definition []byte) ([]byte, error) {
	issuanceProg, err := vmutil.TxMultiSigScript(pubkeys, nrequired)
	if err != nil {
		return nil, err
	}
	builder := vmutil.NewBuilder()
	builder.AddData(definition).AddOp(vm.OP_DROP)
	builder.AddRawBytes(issuanceProg)
	return builder.Program, nil
}

func mapToNullString(in map[string]interface{}) (*sql.NullString, error) {
	var mapJSON []byte
	if len(in) != 0 {
		var err error
		mapJSON, err = json.Marshal(in)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return &sql.NullString{String: string(mapJSON), Valid: len(mapJSON) > 0}, nil
}
