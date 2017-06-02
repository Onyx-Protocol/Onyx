// Package asset maintains a registry of all assets on a
// blockchain.
package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/crypto/sha3"

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
	"chain/protocol/vm/vmutil"
)

const maxAssetCache = 1000

var (
	ErrDuplicateAlias = errors.New("duplicate asset alias")
	ErrBadIdentifier  = errors.New("either ID or alias must be specified, and not both")
)

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
	VMVersion        uint64
	IssuanceProgram  []byte
	InitialBlockHash bc.Hash
	Signer           *signers.Signer
	Tags             map[string]interface{}
	rawDefinition    []byte
	definition       map[string]interface{}
	sortID           string
}

func (asset *Asset) Definition() (map[string]interface{}, error) {
	if asset.definition == nil && len(asset.rawDefinition) > 0 {
		err := json.Unmarshal(asset.rawDefinition, &asset.definition)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return asset.definition, nil
}

func (asset *Asset) RawDefinition() []byte {
	return asset.rawDefinition
}

func (asset *Asset) SetDefinition(def map[string]interface{}) error {
	rawdef, err := serializeAssetDef(def)
	if err != nil {
		return err
	}
	asset.definition = def
	asset.rawDefinition = rawdef
	return nil
}

// Define defines a new Asset.
func (reg *Registry) Define(ctx context.Context, xpubs []chainkd.XPub, quorum int, definition map[string]interface{}, alias string, tags map[string]interface{}, clientToken string) (*Asset, error) {
	assetSigner, err := signers.Create(ctx, reg.db, "asset", xpubs, quorum, clientToken)
	if err != nil {
		return nil, err
	}

	rawDefinition, err := serializeAssetDef(definition)
	if err != nil {
		return nil, errors.Wrap(err, "serializing asset definition")
	}

	path := signers.Path(assetSigner, signers.AssetKeySpace)
	derivedXPubs := chainkd.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	issuanceProgram, vmver, err := multisigIssuanceProgram(derivedPKs, assetSigner.Quorum)
	if err != nil {
		return nil, err
	}

	defhash := bc.NewHash(sha3.Sum256(rawDefinition))
	asset := &Asset{
		definition:       definition,
		rawDefinition:    rawDefinition,
		VMVersion:        vmver,
		IssuanceProgram:  issuanceProgram,
		InitialBlockHash: reg.initialBlockHash,
		AssetID:          bc.ComputeAssetID(issuanceProgram, &reg.initialBlockHash, vmver, &defhash),
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

// UpdateTags modifies the tags of the specified asset. The asset may be
// identified either by id or alias, but not both.
func (reg *Registry) UpdateTags(ctx context.Context, id, alias *string, tags map[string]interface{}) error {
	if (id == nil) == (alias == nil) {
		return errors.Wrap(ErrBadIdentifier)
	}

	// Fetch the existing asset

	var (
		asset *Asset
		err   error
	)

	if id != nil {
		var aid bc.AssetID
		err = aid.UnmarshalText([]byte(*id))
		if err != nil {
			return errors.Wrap(err, "deserialize asset ID")
		}

		asset, err = reg.findByID(ctx, aid)
		if err != nil {
			return errors.Wrap(err, "find asset by ID")
		}
	} else {
		asset, err = reg.FindByAlias(ctx, *alias)
		if err != nil {
			return errors.Wrap(err, "find asset by alias")
		}
	}

	// Revise tags in-memory

	asset.Tags = tags

	// Perform persistent updates

	err = insertAssetTags(ctx, reg.db, asset.AssetID, asset.Tags)
	if err != nil {
		return errors.Wrap(err, "inserting asset tags")
	}

	err = reg.indexAnnotatedAsset(ctx, asset)
	if err != nil {
		return errors.Wrap(err, "update asset index")
	}

	// Revise cache

	reg.cacheMu.Lock()
	reg.cache.Add(asset.AssetID, asset)
	reg.cacheMu.Unlock()

	return nil
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
func (reg *Registry) insertAsset(ctx context.Context, asset *Asset, clientToken string) (*Asset, error) {
	const q = `
		INSERT INTO assets
			(id, alias, signer_id, initial_block_hash, vm_version, issuance_program, definition, client_token)
		VALUES($1::bytea, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (client_token) DO NOTHING
		RETURNING sort_id
  `
	var signerID sql.NullString
	if asset.Signer != nil {
		signerID = sql.NullString{Valid: true, String: asset.Signer.ID}
	}

	nullToken := sql.NullString{
		String: clientToken,
		Valid:  clientToken != "",
	}

	err := reg.db.QueryRowContext(
		ctx, q,
		asset.AssetID, asset.Alias, signerID,
		asset.InitialBlockHash, asset.VMVersion, asset.IssuanceProgram,
		asset.rawDefinition, nullToken,
	).Scan(&asset.sortID)

	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(ErrDuplicateAlias, "an asset with the provided alias already exists")
	} else if err == sql.ErrNoRows && clientToken != "" {
		// There is already an asset with the provided client
		// token. We should return the existing asset.
		asset, err = assetByClientToken(ctx, reg.db, clientToken)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving existing asset")
		}
	} else if err != nil {
		return nil, errors.Wrap(err)
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
	_, err = db.ExecContext(ctx, q, assetID, tagsParam)
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
		SELECT assets.id, assets.alias, assets.vm_version, assets.issuance_program, assets.definition,
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
		signerID   sql.NullString
		signerType string
		quorum     int
		keyIndex   uint64
		xpubs      [][]byte
		tags       []byte
	)
	err := db.QueryRowContext(ctx, fmt.Sprintf(baseQ, pred), args...).Scan(
		&a.AssetID,
		&a.Alias,
		&a.VMVersion,
		&a.IssuanceProgram,
		&a.rawDefinition,
		&a.InitialBlockHash,
		&a.sortID,
		&signerID,
		&signerType,
		(*pq.ByteaArray)(&xpubs),
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

	if alias.Valid {
		a.Alias = &alias.String
	}

	if len(tags) > 0 {
		err := json.Unmarshal(tags, &a.Tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	if len(a.rawDefinition) > 0 {
		// ignore errors; non-JSON asset definitions can still end up
		// on the blockchain from non-Chain Core clients.
		_ = json.Unmarshal(a.rawDefinition, &a.definition)
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
	if def == nil {
		return []byte{}, nil
	}
	return json.MarshalIndent(def, "", "  ")
}

func multisigIssuanceProgram(pubkeys []ed25519.PublicKey, nrequired int) (program []byte, vmversion uint64, err error) {
	issuanceProg, err := vmutil.P2SPMultiSigProgram(pubkeys, nrequired)
	if err != nil {
		return nil, 0, err
	}
	builder := vmutil.NewBuilder()
	builder.AddRawBytes(issuanceProg)
	prog, err := builder.Build()
	return prog, 1, err
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
