package core

import (
	"encoding/json"
	"sync"

	"golang.org/x/net/context"

	"chain/core/pb"
	"chain/core/query"
	"chain/core/signers"
	"chain/crypto/ed25519/chainkd"
	cjson "chain/encoding/json"
	"chain/net/http/httpjson"
	"chain/net/http/reqid"
	"chain/protocol/bc"
)

// This type enforces JSON field ordering in API output.
type assetResponse struct {
	ID              bc.AssetID      `json:"id"`
	Alias           string          `json:"alias"`
	VMVersion       uint64          `json:"vm_version"`
	IssuanceProgram cjson.HexBytes  `json:"issuance_program"`
	Keys            []*assetKey     `json:"keys"`
	Quorum          int             `json:"quorum"`
	Definition      json.RawMessage `json:"definition"`
	RawDefinition   cjson.HexBytes  `json:"raw_definition"`
	Tags            json.RawMessage `json:"tags"`
	IsLocal         query.Bool      `json:"is_local"`
}

type assetKey struct {
	RootXPub            chainkd.XPub     `json:"root_xpub"`
	AssetPubkey         cjson.HexBytes   `json:"asset_pubkey"`
	AssetDerivationPath []cjson.HexBytes `json:"asset_derivation_path"`
}

func (h *Handler) CreateAssets(ctx context.Context, in *pb.CreateAssetsRequest) (*pb.CreateAssetsResponse, error) {
	responses := make([]*pb.CreateAssetsResponse_Response, len(in.Requests))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(func(err error) {
				responses[i] = &pb.CreateAssetsResponse_Response{
					Error: protobufErr(err),
				}
			})

			var tags, def map[string]interface{}
			if len(in.Requests[i].Tags) > 0 {
				err := json.Unmarshal(in.Requests[i].Tags, &tags)
				if err != nil {
					responses[i] = &pb.CreateAssetsResponse_Response{
						Error: protobufErr(httpjson.ErrBadRequest),
					}
					return
				}
			}
			if len(in.Requests[i].Definition) > 0 {
				err := json.Unmarshal(in.Requests[i].Definition, &def)
				if err != nil {
					responses[i] = &pb.CreateAssetsResponse_Response{
						Error: protobufErr(httpjson.ErrBadRequest),
					}
					return
				}
			}

			xpubs, err := bytesToKeys(in.Requests[i].RootXpubs)
			if err != nil {
				responses[i] = &pb.CreateAssetsResponse_Response{
					Error: protobufErr(err),
				}
				return
			}

			asset, err := h.Assets.Define(
				subctx,
				xpubs,
				int(in.Requests[i].Quorum),
				def,
				in.Requests[i].Alias,
				tags,
				in.Requests[i].ClientToken,
			)
			if err != nil {
				responses[i] = &pb.CreateAssetsResponse_Response{
					Error: protobufErr(err),
				}
				return
			}
			var keys []*pb.Asset_Key
			for _, xpub := range asset.Signer.XPubs {
				path := signers.Path(asset.Signer, signers.AssetKeySpace)
				derived := xpub.Derive(path)
				keys = append(keys, &pb.Asset_Key{
					AssetPubkey:         derived[:],
					RootXpub:            xpub[:],
					AssetDerivationPath: path,
				})
			}

			var aliasStr string
			if asset.Alias != nil {
				aliasStr = *asset.Alias
			}

			responses[i] = &pb.CreateAssetsResponse_Response{
				Asset: &pb.Asset{
					Id:              asset.AssetID[:],
					Alias:           aliasStr,
					VmVersion:       asset.VMVersion,
					IssuanceProgram: asset.IssuanceProgram,
					Keys:            keys,
					Quorum:          int32(asset.Signer.Quorum),
					Definition:      in.Requests[i].Definition,
					Tags:            in.Requests[i].Tags,
					IsLocal:         true,
				},
			}
		}(i)
	}

	wg.Wait()
	return &pb.CreateAssetsResponse{Responses: responses}, nil
}
