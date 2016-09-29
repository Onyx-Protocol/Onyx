package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"chain/core/signers"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
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
	AssetID          bc.AssetID
	Alias            *string
	Definition       map[string]interface{}
	IssuanceProgram  []byte
	InitialBlockHash bc.Hash
	Signer           *signers.Signer
	Tags             map[string]interface{}
	sortID           string
}

// Define defines a new Asset.
func Define(ctx context.Context, xpubs []string, quorum int, definition map[string]interface{}, initialBlockHash bc.Hash, alias string, tags map[string]interface{}, clientToken *string) (*Asset, error) {
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

	derivedXPubs := chainkd.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	issuanceProgram, err := programWithDefinition(derivedPKs, assetSigner.Quorum, serializedDef)
	if err != nil {
		return nil, err
	}

	asset := &Asset{
		Definition:       definition,
		IssuanceProgram:  issuanceProgram,
		InitialBlockHash: initialBlockHash,
		AssetID:          bc.ComputeAssetID(issuanceProgram, initialBlockHash, 1),
		Signer:           assetSigner,
		Tags:             tags,
	}
	if alias != "" {
		asset.Alias = &alias
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

// FindByID retrieves an Asset record along with its signer, given an assetID.
func FindByID(ctx context.Context, id bc.AssetID) (*Asset, error) {
	return lookupAsset(ctx, id, "")
}

// FindByAlias retrieves an Asset record along with its signer,
// given an asset alias.
func FindByAlias(ctx context.Context, alias string) (*Asset, error) {
	return lookupAsset(ctx, bc.AssetID{}, alias)
}

// Archive marks an Asset record as archived, effectively "deleting" it.
func Archive(ctx context.Context, id bc.AssetID, alias string) error {
	asset, err := lookupAsset(ctx, id, alias)
	if err != nil {
		return errors.Wrap(err, "asset is missing")
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

	if asset.Signer != nil {
		err = signers.Archive(ctx, "asset", asset.Signer.ID)
		if err != nil {
			return errors.Wrap(err, "archive asset signer")
		}
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return errors.Wrap(err, "committing archive asset dbtx")
	}

	return nil
}

// insertAsset adds the asset to the database. If the asset has a client token,
// and there already exists an asset with that client token, insertAsset will
// lookup and return the existing asset instead.
func insertAsset(ctx context.Context, asset *Asset, clientToken *string) (*Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
    INSERT INTO assets
	 	(id, alias, signer_id, initial_block_hash, issuance_program, definition, client_token)
    VALUES($1, $2, $3, $4, $5, $6, $7)
    ON CONFLICT (client_token) DO NOTHING
	RETURNING sort_id
  `
	defParams, err := mapToNullString(asset.Definition)
	if err != nil {
		return nil, err
	}

	var signerID sql.NullString
	if asset.Signer != nil {
		signerID = sql.NullString{Valid: true, String: asset.Signer.ID}
	}

	err = pg.QueryRow(
		ctx, q,
		asset.AssetID, asset.Alias, signerID,
		asset.InitialBlockHash, asset.IssuanceProgram,
		defParams, clientToken,
	).Scan(&asset.sortID)

	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "non-unique alias")
	} else if err == sql.ErrNoRows && clientToken != nil {
		// There is already an asset with the provided client
		// token. We should return the existing asset.
		asset, err = assetByClientToken(ctx, *clientToken)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving existing asset")
		}
	} else if err != nil {
		return nil, err
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

func lookupAsset(ctx context.Context, idQ bc.AssetID, aliasQ string) (*Asset, error) {
	if idQ != (bc.AssetID{}) && aliasQ != "" {
		return nil, errors.New("cannot refer to asset by both ID and alias")
	}

	return assetQuery(ctx, "id=$1 OR ($2!='' AND alias=$2)", idQ, aliasQ)
}

// assetByClientToken loads an asset from the database using its client token.
func assetByClientToken(ctx context.Context, clientToken string) (*Asset, error) {
	return assetQuery(ctx, "client_token=$1", clientToken)
}

func assetQuery(ctx context.Context, pred string, args ...interface{}) (*Asset, error) {
	const baseQ = `
		SELECT id, alias, issuance_program, definition,
			initial_block_hash, signer_id, archived, sort_id
		FROM assets
		WHERE %s
		LIMIT 1
	`
	var (
		a          Asset
		alias      sql.NullString
		archived   bool
		signerID   sql.NullString
		definition []byte
	)
	err := pg.QueryRow(ctx, fmt.Sprintf(baseQ, pred), args...).Scan(
		&a.AssetID,
		&alias,
		&a.IssuanceProgram,
		&definition,
		&a.InitialBlockHash,
		&signerID,
		&archived,
		&a.sortID,
	)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	} else if err != nil {
		return nil, err
	}
	if archived {
		return nil, ErrArchived
	}

	if signerID.Valid {
		// Only try to fetch the signer if this is a
		// local asset.
		sig, err := signers.Find(ctx, "asset", signerID.String)
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
		a.Alias = &alias.String
	}

	const tagQ = `SELECT tags FROM asset_tags WHERE asset_id=$1`
	var tags []byte
	err = pg.QueryRow(ctx, tagQ, a.AssetID.String()).Scan(&tags)
	if err != nil && err != sql.ErrNoRows {
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
	issuanceProg, err := vmutil.P2SPMultiSigProgram(pubkeys, nrequired)
	if err != nil {
		return nil, err
	}
	builder := vmutil.NewBuilder()
	builder.AddData(definition).AddOp(vm.OP_DROP)
	builder.AddRawBytes(issuanceProg)
	return builder.Program, nil
}

func definitionFromProgram(program []byte) ([]byte, error) {
	pops, err := vm.ParseProgram(program)
	if err != nil {
		return nil, err
	}
	if len(pops) < 2 {
		return nil, errors.New("bad issuance program")
	}
	if pops[1].Op != vm.OP_DROP {
		return nil, errors.New("bad issuance program")
	}
	return pops[0].Data, nil
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
