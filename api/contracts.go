package api

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"

	"chain/api/smartcontracts/orderbook"
	"chain/api/smartcontracts/voting"
	"chain/api/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/net/http/httpjson"
)

const (
	votingRightsPageSize  = 100
	votingTalliesPageSize = 50
)

type globalFindOrder struct {
	OfferedAssetIDs []bc.AssetID `json:"offered_asset_ids"`
	PaymentAssetIDs []bc.AssetID `json:"payment_asset_ids"`
}

func findOrders(ctx context.Context, req globalFindOrder) ([]*orderbook.OpenOrder, error) {
	qvals := httpjson.Request(ctx).URL.Query()
	if status, ok := qvals["status"]; !ok || status[0] != "open" {
		// TODO(tessr): find closed orders
		return nil, errors.Wrap(httpjson.ErrBadRequest, "unimplemented: find all orders")
	}
	orders, err := orderbook.FindOpenOrders(ctx, req.OfferedAssetIDs, req.PaymentAssetIDs)
	if err != nil {
		return nil, errors.Wrap(err, "finding orders by offered and payment asset ids")
	}

	return orders, nil
}

func findAccountOrders(ctx context.Context, accountID string) ([]*orderbook.OpenOrder, error) {
	qvals := httpjson.Request(ctx).URL.Query()
	if status, ok := qvals["status"]; !ok || status[0] != "open" {
		// TODO(tessr): find closed orders
		return nil, errors.Wrap(httpjson.ErrBadRequest, "unimplemented: find all orders")
	}
	if aids, ok := qvals["asset_id"]; ok {
		var assetIDs []bc.AssetID
		for _, id := range aids {
			var assetID bc.AssetID
			err := assetID.UnmarshalText([]byte(id))
			if err != nil {
				return nil, errors.Wrap(httpjson.ErrBadRequest, "invalid assetID")
			}
			assetIDs = append(assetIDs, assetID)
		}
		orders, err := orderbook.FindOpenOrdersBySellerAndAsset(ctx, accountID, assetIDs)
		if err != nil {
			return nil, errors.Wrap(err, "finding orders by seller and asset")
		}
		return orders, nil
	}
	orders, err := orderbook.FindOpenOrdersBySeller(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func findAccountVotingRights(ctx context.Context, accountID string) (map[string]interface{}, error) {
	prev, limit, err := getPageData(ctx, votingRightsPageSize)
	if err != nil {
		return nil, err
	}

	rightsWithUTXOs, last, err := voting.FindRightsForAccount(ctx, accountID, prev, limit)
	if err != nil {
		return nil, errors.Wrap(err, "finding account voting rights")
	}

	rights := make([]map[string]interface{}, 0, len(rightsWithUTXOs))
	for _, r := range rightsWithUTXOs {
		var actionTypes []string
		if r.Outpoint.Hash == r.UTXO.Hash && r.Outpoint.Index == r.UTXO.Index {
			actionTypes = append(actionTypes, "votingright-authenticate", "votingright-transfer", "votingright-delegate")
		} else {
			actionTypes = append(actionTypes, "votingright-recall")
		}

		rightToken := map[string]interface{}{
			"asset_id":       r.AssetID,
			"action_types":   actionTypes,
			"transaction_id": r.UTXO.Hash,
			"index":          r.UTXO.Index,
		}
		rights = append(rights, rightToken)
	}
	return map[string]interface{}{
		"balances": rights,
		"last":     last,
	}, nil
}

func getVotingRightHistory(ctx context.Context, assetID string) ([]map[string]interface{}, error) {
	var parsedAssetID bc.AssetID
	err := parsedAssetID.UnmarshalText([]byte(assetID))
	if err != nil {
		return nil, errors.Wrap(err, "parsing asset ID")
	}

	rightsWithUTXOs, err := voting.FindRightsForAsset(ctx, parsedAssetID)
	if err != nil {
		return nil, err
	}

	rights := make([]map[string]interface{}, 0, len(rightsWithUTXOs))
	for _, r := range rightsWithUTXOs {
		right := map[string]interface{}{
			"asset_id":       r.AssetID,
			"account_id":     r.AccountID,
			"holder":         chainjson.HexBytes(r.HolderScript),
			"transferable":   r.Delegatable,
			"deadline":       time.Unix(r.Deadline, 0).Format(time.RFC3339),
			"transaction_id": r.Outpoint.Hash,
			"index":          r.Outpoint.Index,
		}
		rights = append(rights, right)
	}
	return rights, nil
}

type votingTallyRequest struct {
	VotingTokenAssetIDs []bc.AssetID `json:"asset_ids"`
	After               string       `json:"after,omitempty"`
}

// POST /v3/contracts/voting-tokens/tally
func getVotingTokenTally(ctx context.Context, req votingTallyRequest) (map[string]interface{}, error) {
	// TODO(jackson): Avoid calling voting.TallyVotes separately for each
	// asset ID by modifying voting.TallyVotes() to query multiple asset IDs
	// at once.
	// TODO(jackson): Add real pagination. For now, we fake it.

	last := ""
	tallies := make([]voting.Tally, 0, len(req.VotingTokenAssetIDs))
	for _, assetID := range req.VotingTokenAssetIDs {
		if req.After != "" && assetID.String() <= req.After {
			continue
		}

		tally, err := voting.TallyVotes(ctx, assetID)
		if err != nil {
			return nil, err
		}
		last = tally.AssetID.String()
		tallies = append(tallies, tally)

		if len(tallies) >= votingTalliesPageSize {
			break
		}
	}

	return map[string]interface{}{
		"votingtokens": tallies,
		"last":         last,
	}, nil
}

type votingToken struct {
	token bc.AssetID
	right bc.AssetID
}

// parseVotingBuildRequest parses `votingright` and `voting` BuildRequest
// sources and destinations. Unlike other asset types, voting request inputs
// and outputs need data from each other in order to build the correct
// txbuilder.Reservers and txbuilder.Receivers.
func parseVotingBuildRequest(ctx context.Context, sources []*Source, destinations []*Destination) (
	srcs []*txbuilder.Source,
	dsts []*txbuilder.Destination,
	err error,
) {
	var (
		rightSrcsByAssetID  = map[bc.AssetID]*Source{}
		rightDstsByAssetID  = map[bc.AssetID]*Destination{}
		tokenSrcsByAssetIDs = map[votingToken]*Source{}
		tokenDstsByAssetIDs = map[votingToken]*Destination{}
	)

	// Pair sources and destinations by asset id, and split them into voting
	// rights and voting tokens.
	for _, src := range sources {
		if src.AssetID == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "asset type unspecified")
		}
		if strings.HasPrefix(src.Type, "votingright-") {
			rightSrcsByAssetID[*src.AssetID] = src
			continue
		}
		if src.VotingRight == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "voting sources must include voting_right_asset_id")
		}
		vt := votingToken{token: *src.AssetID, right: *src.VotingRight}
		tokenSrcsByAssetIDs[vt] = src
	}
	for _, dst := range destinations {
		if dst.AssetID == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "asset type unspecified")
		}
		if dst.Type == "votingright" {
			rightDstsByAssetID[*dst.AssetID] = dst
			continue
		}
		if dst.VotingRight == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "voting destinations must include voting_right_asset_id")
		}
		vt := votingToken{token: *dst.AssetID, right: *dst.VotingRight}
		tokenDstsByAssetIDs[vt] = dst
	}

	// Parse the voting rights first. Some voting token clauses require
	// knowledge of the voting right script.
	srcs, dsts, rightsByAssetID, err := parseVotingRights(ctx, rightSrcsByAssetID, rightDstsByAssetID)
	if err != nil {
		return nil, nil, err
	}

	// Parse the voting tokens.
	for assetIDs, dst := range tokenDstsByAssetIDs {
		src, ok := tokenSrcsByAssetIDs[assetIDs]
		if !ok {
			// If there's no corresponding source with a `voting`
			// type, then the destination should be a voting token
			// issuance.
			if dst.VotingRight == nil {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "new voting tokens must provide corresponding voting right")
			}
			if dst.AdminScript == nil {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "new voting tokens must provide the voting system admin script")
			}
			if dst.Options <= 0 {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "new voting tokens must have 1 or more voting options")
			}
			dsts = append(dsts, &txbuilder.Destination{
				AssetAmount: bc.AssetAmount{AssetID: assetIDs.token, Amount: dst.Amount},
				Metadata:    dst.Metadata,
				Receiver:    voting.TokenIssuance(ctx, *dst.VotingRight, dst.AdminScript, dst.Options, dst.SecretHash),
			})
			continue
		}

		token, err := voting.FindTokenForAsset(ctx, assetIDs.token, assetIDs.right)
		if err != nil {
			return nil, nil, err
		}
		if token == nil {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "unknown voting token")
		}

		var (
			reserver txbuilder.Reserver
			receiver txbuilder.Receiver
		)
		switch src.Type {
		case "voting-intent":
			if !token.State.Distributed() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting token must be in distributed state")
			}
			if token.State.Finished() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting has been closed")
			}
			reserver, receiver, err = voting.TokenIntent(ctx, token, rightsByAssetID[token.Right])
			if err != nil {
				return nil, nil, err
			}
		case "voting-vote":
			if !token.State.Intended() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting token must be in intended state")
			}
			if token.State.Finished() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting has been closed")
			}
			reserver, receiver, err = voting.TokenVote(ctx, token, rightsByAssetID[token.Right], dst.Vote, dst.QuorumSecret)
			if err != nil {
				return nil, nil, err
			}
		case "voting-close":
			if token.State.Finished() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting has already been closed")
			}
			reserver, receiver, err = voting.TokenFinish(ctx, token)
			if err != nil {
				return nil, nil, err
			}
		default:
			// TODO(jackson): Implement all other voting token clauses
			return nil, nil, fmt.Errorf("unimplemented src.type: %s", src.Type)
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: assetIDs.token, Amount: src.Amount},
			Reserver:    reserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: assetIDs.token, Amount: dst.Amount},
			Metadata:    dst.Metadata,
			Receiver:    receiver,
		})
	}

	return srcs, dsts, nil
}

// parseVotingRights will pair the vrtoken sources and destinations up by
// asset ID, and use the information from both to construct the
// txbuilder.Sources and txbuilder.Destinations.
func parseVotingRights(ctx context.Context, srcsByAssetID map[bc.AssetID]*Source, dstsByAssetID map[bc.AssetID]*Destination) (
	srcs []*txbuilder.Source,
	dsts []*txbuilder.Destination,
	byAssetID map[bc.AssetID]txbuilder.Receiver,
	err error,
) {
	byAssetID = map[bc.AssetID]txbuilder.Receiver{}

	if len(srcsByAssetID) > len(dstsByAssetID) {
		// Both the source and destination must be provided in the same build
		// request. This is unavoidable because:
		// - the output contract script requires knowledge of the input's chain of ownership
		// - the sigscript needs to provide the new contract parameters
		// The only exception is issuing new voting right tokens. Then there
		// will be more destinations than sources.
		return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest,
			"voting right source and destinations must be provided in the same build request")
	}

	for assetID, dst := range dstsByAssetID {
		// Validate the destination.
		if dst.Amount != 0 {
			return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting right destinations do not take amounts")
		}

		src, ok := srcsByAssetID[assetID]
		if !ok {
			// If there is no votingright source, then assume this is an attempt
			// to issue into a new asset into a voting right contract.
			if dst.AdminScript == nil {
				return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting right issuance requires a voting system admin script")
			}
			holder, err := dst.buildAddress(ctx)
			if err != nil {
				return nil, nil, nil, err
			}
			receiver := voting.RightIssuance(ctx, dst.AdminScript, holder)
			dsts = append(dsts, &txbuilder.Destination{
				AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 1},
				Metadata:    dst.Metadata,
				Receiver:    receiver,
			})
			byAssetID[assetID] = receiver
			continue
		}

		// Validate the source.
		if src.TxHash == nil {
			src.TxHash = src.TxHashAsID
		}
		if (src.TxHash == nil || src.Index == nil) && src.AccountID == "" {
			return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "bad voting right source")
		}
		if src.Amount != 0 {
			return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting right sources do not take amounts")
		}

		// Lookup the voting right by the asset ID. We'll need some of its
		// script data, such as the previous chain of ownership.
		old, err := voting.FindRightUTXO(ctx, *src.AssetID)
		if err == pg.ErrUserInputNotFound {
			return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "bad voting right source")
		} else if err != nil {
			return nil, nil, nil, err
		}

		// If a src account ID was provided, ensure that it matches the current utxo.
		if src.AccountID != "" && (old.AccountID == nil || *old.AccountID != src.AccountID) {
			return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "bad voting right source")
		}
		// If tx_hash and index were provided, ensure that they match the current utxo.
		if (src.TxHash != nil && src.Index != nil) &&
			(bc.Outpoint{Hash: *src.TxHash, Index: *src.Index}) != old.Outpoint {
			return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "bad voting right source")
		}

		var (
			reserver txbuilder.Reserver
			receiver txbuilder.Receiver
		)
		switch src.Type {
		case "votingright-authenticate":
			reserver, receiver, err = voting.RightAuthentication(ctx, old)
			if err != nil {
				return nil, nil, nil, err
			}
		case "votingright-transfer":
			script, err := dst.buildAddress(ctx)
			if err != nil {
				return nil, nil, nil, err
			}
			reserver, receiver, err = voting.RightTransfer(ctx, old, script)
			if err != nil {
				return nil, nil, nil, err
			}
		case "votingright-delegate":
			if !old.Delegatable {
				return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "delegating this voting right is prohibited")
			}
			if dst.Deadline.Unix() > old.Deadline {
				return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "cannot extend deadline beyond current deadline")
			}
			script, err := dst.buildAddress(ctx)
			if err != nil {
				return nil, nil, nil, err
			}
			var (
				delegatable = old.Delegatable
				deadline    = old.Deadline
			)
			if dst.Transferable != nil {
				delegatable = *dst.Transferable
			}
			if !dst.Deadline.IsZero() {
				deadline = dst.Deadline.Unix()
			}
			reserver, receiver, err = voting.RightDelegation(ctx, old, script, deadline, delegatable)
			if err != nil {
				return nil, nil, nil, err
			}
		case "votingright-recall":
			claims, err := voting.FindRightsForAsset(ctx, assetID)
			if err != nil {
				return nil, nil, nil, err
			}
			if len(claims) < 2 {
				// You need at least two claims to have a recallable voting right.
				return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "bad voting right source")
			}

			// Find the earliest, active voting right claim that this account
			// has on this voting right token. We'll recall back to that point.
			var (
				recallPoint    *voting.RightWithUTXO
				recallPointIdx int
			)
			for i, claim := range claims {
				if claim.AccountID != nil && *claim.AccountID == dst.AccountID {
					recallPoint = claim
					recallPointIdx = i
					break
				}
			}
			if recallPoint == nil {
				return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting right not recallable")
			}
			reserver, receiver, err = voting.RightRecall(ctx, old, recallPoint, claims[recallPointIdx+1:len(claims)-1])
			if err != nil {
				return nil, nil, nil, err
			}
		default:
			return nil, nil, nil, errors.WithDetailf(ErrBadBuildRequest, "`%s` source type unimplemented", src.Type)
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 1},
			Reserver:    reserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 1},
			Metadata:    dst.Metadata,
			Receiver:    receiver,
		})
		byAssetID[assetID] = receiver
	}
	return srcs, dsts, byAssetID, nil
}
