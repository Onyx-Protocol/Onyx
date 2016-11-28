// Package asset maintains a registry of all assets on a
// blockchain.
package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"
	"github.com/lib/pq"

	"chain/core/pin"
	"chain/core/signers"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

const maxAssetCache = 1000

var ErrDuplicateAlias = errors.New("duplicate asset alias")

func NewRegistry(db pg.DB, chain *protocol.Chain, pinStore *pin.Store) *Registry {
	return &Registry{
		db:               db,
		chain:            chain,
		initialBlockHash: chain.InitialBlockHash,
		pinStore:         pinStore,
		cache:            lru.New(maxAssetCache),
		aliasCache:       lru.New(maxAssetCache),
	}
}

// Registry tracks and stores all known assets on a blockchain.
type Registry struct {
	db               pg.DB
	chain            *protocol.Chain
	indexer          Saver
	initialBlockHash bc.Hash
	pinStore         *pin.Store

	idGroup    singleflight.Group
	aliasGroup singleflight.Group

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache
}

func (reg *Registry) IndexAssets(indexer Saver) {
	reg.indexer = indexer
}

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
func (reg *Registry) Define(ctx context.Context, xpubs []string, quorum int, definition map[string]interface{}, alias string, tags map[string]interface{}, clientToken *string) (*Asset, error) {
	assetSigner, err := signers.Create(ctx, reg.db, "asset", xpubs, quorum, clientToken)
	if err != nil {
		return nil, err
	}

	serializedDef, err := serializeAssetDef(definition)
	if err != nil {
		return nil, errors.Wrap(err, "serializing asset definition")
	}

	path := signers.Path(assetSigner, signers.AssetKeySpace)
	derivedXPubs := chainkd.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	issuanceProgram, err := programWithDefinition(derivedPKs, assetSigner.Quorum, serializedDef)
	if err != nil {
		return nil, err
	}

	asset := &Asset{
		Definition:       definition,
		IssuanceProgram:  issuanceProgram,
		InitialBlockHash: reg.initialBlockHash,
		AssetID:          bc.ComputeAssetID(issuanceProgram, reg.initialBlockHash, 1),
		Signer:           assetSigner,
		Tags:             tags,
	}
	if alias != "" {
		asset.Alias = &alias
	}

	asset, err = reg.insertAsset(ctx, asset, clientToken)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset")
	}

	err = insertAssetTags(ctx, reg.db, asset.AssetID, tags)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset tags")
	}

	err = reg.indexAnnotatedAsset(ctx, asset)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated asset")
	}

	return asset, nil
}

// findByID retrieves an Asset record along with its signer, given an assetID.
func (reg *Registry) findByID(ctx context.Context, id bc.AssetID) (*Asset, error) {
	reg.cacheMu.Lock()
	cached, ok := reg.cache.Get(id)
	reg.cacheMu.Unlock()
	if ok {
		return cached.(*Asset), nil
	}

	untypedAsset, err := reg.idGroup.Do(id.String(), func() (interface{}, error) {
		return assetQuery(ctx, reg.db, "assets.id=$1", id)
	})
	if err != nil {
		return nil, err
	}

	asset := untypedAsset.(*Asset)
	reg.cacheMu.Lock()
	reg.cache.Add(id, asset)
	reg.cacheMu.Unlock()
	return asset, nil
}

// FindByAlias retrieves an Asset record along with its signer,
// given an asset alias.
func (reg *Registry) FindByAlias(ctx context.Context, alias string) (*Asset, error) {
	reg.cacheMu.Lock()
	cachedID, ok := reg.aliasCache.Get(alias)
	reg.cacheMu.Unlock()
	if ok {
		return reg.findByID(ctx, cachedID.(bc.AssetID))
	}

	untypedAsset, err := reg.aliasGroup.Do(alias, func() (interface{}, error) {
		asset, err := assetQuery(ctx, reg.db, "assets.alias=$1", alias)
		return asset, err
	})
	if err != nil {
		return nil, err
	}

	a := untypedAsset.(*Asset)
	reg.cacheMu.Lock()
	reg.aliasCache.Add(alias, a.AssetID)
	reg.cache.Add(a.AssetID, a)
	reg.cacheMu.Unlock()
	return a, nil

}

// insertAsset adds the asset to the database. If the asset has a client token,
// and there already exists an asset with that client token, insertAsset will
// lookup and return the existing asset instead.
func (reg *Registry) insertAsset(ctx context.Context, asset *Asset, clientToken *string) (*Asset, error) {
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

	err = reg.db.QueryRow(
		ctx, q,
		asset.AssetID, asset.Alias, signerID,
		asset.InitialBlockHash, asset.IssuanceProgram,
		defParams, clientToken,
	).Scan(&asset.sortID)

	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(ErrDuplicateAlias, "an asset with the provided alias already exists")
	} else if err == sql.ErrNoRows && clientToken != nil {
		// There is already an asset with the provided client
		// token. We should return the existing asset.
		asset, err = assetByClientToken(ctx, reg.db, *clientToken)
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
func insertAssetTags(ctx context.Context, db pg.DB, assetID bc.AssetID, tags map[string]interface{}) error {
	tagsParam, err := mapToNullString(tags)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `
		INSERT INTO asset_tags (asset_id, tags) VALUES ($1, $2)
		ON CONFLICT (asset_id) DO UPDATE SET tags = $2
	`
	_, err = db.Exec(ctx, q, assetID.String(), tagsParam)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

// assetByClientToken loads an asset from the database using its client token.
func assetByClientToken(ctx context.Context, db pg.DB, clientToken string) (*Asset, error) {
	return assetQuery(ctx, db, "assets.client_token=$1", clientToken)
}

func assetQuery(ctx context.Context, db pg.DB, pred string, args ...interface{}) (*Asset, error) {
	const baseQ = `
		SELECT assets.id, assets.alias, assets.issuance_program, assets.definition,
			assets.initial_block_hash, assets.sort_id,
			signers.id, COALESCE(signers.type, ''), COALESCE(signers.xpubs, '{}'),
			COALESCE(signers.quorum, 0), COALESCE(signers.key_index, 0),
			asset_tags.tags
		FROM assets
		LEFT JOIN signers ON signers.id=assets.signer_id
		LEFT JOIN asset_tags ON asset_tags.asset_id=assets.id
		WHERE %s
		LIMIT 1
	`
	var (
		a          Asset
		alias      sql.NullString
		definition []byte
		signerID   sql.NullString
		signerType string
		quorum     int
		keyIndex   uint64
		xpubs      []string
		tags       []byte
	)
	err := db.QueryRow(ctx, fmt.Sprintf(baseQ, pred), args...).Scan(
		&a.AssetID,
		&a.Alias,
		&a.IssuanceProgram,
		&definition,
		&a.InitialBlockHash,
		&a.sortID,
		&signerID,
		&signerType,
		(*pq.StringArray)(&xpubs),
		&quorum,
		&keyIndex,
		&tags,
	)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	} else if err != nil {
		return nil, err
	}

	if signerID.Valid {
		a.Signer, err = signers.New(signerID.String, signerType, xpubs, quorum, keyIndex)
		if err != nil {
			return nil, err
		}
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
