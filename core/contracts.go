package core

import (
	"math"

	"golang.org/x/net/context"

	"chain/core/issuer"
	"chain/core/smartcontracts/orderbook"
	"chain/core/smartcontracts/voting"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/net/http/httpjson"
)

const (
	votingRightsPageSize  = 100
	votingTokensPageSize  = 100
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

	accRights, holders, last, err := voting.FindRightsForAccount(ctx, accountID, prev, limit)
	if err != nil {
		return nil, errors.Wrap(err, "finding account voting rights")
	}

	rights := make([]map[string]interface{}, 0, len(accRights))
	for _, r := range accRights {
		var actionTypes []string
		if accountID == holders[r.AssetID] {
			actionTypes = append(actionTypes, "authenticate-voting-right", "transfer-voting-right")
			if r.Delegatable {
				actionTypes = append(actionTypes, "delegate-voting-right")
			}
		} else {
			actionTypes = append(actionTypes, "recall-voting-right")
		}

		rightToken := map[string]interface{}{
			"voting_right_asset_id": r.AssetID,
			"action_types":          actionTypes,
			"holding_account_id":    holders[r.AssetID],
		}
		rights = append(rights, rightToken)
	}
	return map[string]interface{}{
		"balances": rights,
		"last":     last,
	}, nil
}

func getVotingRightOwners(ctx context.Context, assetID string) (map[string]interface{}, error) {
	var parsedAssetID bc.AssetID
	err := parsedAssetID.UnmarshalText([]byte(assetID))
	if err != nil {
		return nil, errors.Wrap(err, "parsing asset ID")
	}

	// This endpoint isn't paginated, but uses an envelope like other
	// paginated endpoints. This gives us the flexibility to paginate
	// it in the future.
	prev, _, err := getPageData(ctx, votingRightsPageSize)
	if err != nil {
		return nil, err
	}

	rightsWithUTXOs := []*voting.Right{}
	if prev == "" {
		rightsWithUTXOs, err = voting.FindRightsForAsset(ctx, parsedAssetID)
		if err != nil {
			return nil, err
		}
	}

	last := ""
	rights := make([]map[string]interface{}, 0, len(rightsWithUTXOs))
	for _, r := range rightsWithUTXOs {
		right := map[string]interface{}{
			"voting_right_asset_id": r.AssetID,
			"account_id":            r.AccountID,
			"holder_script":         chainjson.HexBytes(r.HolderScript),
			"transferable":          r.Delegatable, // DEPRECATED
			"can_delegate":          r.Delegatable,
			"transaction_id":        r.Outpoint.Hash,
			"index":                 r.Outpoint.Index,
		}
		rights = append(rights, right)
		last = r.Outpoint.Hash.String()
	}
	return map[string]interface{}{
		"owners": rights,
		"last":   last,
	}, nil
}

type votingTallyRequest struct {
	VotingTokenAssetIDs []bc.AssetID `json:"voting_token_asset_ids"`
}

// POST /v3/contracts/voting-tokens/tally
func getVotingTokenTally(ctx context.Context, req votingTallyRequest) (map[string]interface{}, error) {
	// TODO(jackson): Avoid calling voting.TallyVotes separately for each
	// asset ID by modifying voting.TallyVotes() to query multiple asset IDs
	// at once.
	// TODO(jackson): Add real pagination. For now, we fake it.

	prev, _, err := getPageData(ctx, votingTalliesPageSize)
	if err != nil {
		return nil, err
	}

	last := ""
	tallies := make([]voting.Tally, 0, len(req.VotingTokenAssetIDs))
	if prev == "" {
		for _, assetID := range req.VotingTokenAssetIDs {
			tally, err := voting.TallyVotes(ctx, assetID)
			if err != nil {
				return nil, err
			}
			last = tally.AssetID.String()
			tallies = append(tallies, tally)
		}
	}

	return map[string]interface{}{
		"tallies": tallies,
		"last":    last,
	}, nil
}

// POST /v3/contracts/voting-tokens/votes
func getVotingTokenVotes(ctx context.Context, req struct {
	AssetIDs  []bc.AssetID `json:"voting_token_asset_ids"`
	AccountID string       `json:"account_id,omitempty"` // optional
}) (map[string]interface{}, error) {
	prev, limit, err := getPageData(ctx, votingTokensPageSize)
	if err != nil {
		return nil, err
	}

	tokens, last, err := voting.GetVotes(ctx, req.AssetIDs, req.AccountID, prev, limit)
	if err != nil {
		return nil, errors.Wrap(err, "getting voting token votes")
	}

	votes := []map[string]interface{}{}
	for _, t := range tokens {
		actionTypes := []string{}
		if t.State.Open() {
			switch {
			case t.State.Distributed():
				actionTypes = append(actionTypes, "register-voting-token", "redistribute-voting-token")
			case t.State.Registered(), t.State.Voted():
				actionTypes = append(actionTypes, "vote")
			}
			actionTypes = append(actionTypes, "close-vote")
			actionTypes = append(actionTypes, "invalidate-vote")
		}

		votes = append(votes, map[string]interface{}{
			"transaction_id":        t.Outpoint.Hash,
			"transaction_output":    t.Outpoint.Index,
			"voting_token_asset_id": t.AssetID,
			"voting_right_asset_id": t.Right,
			"amount":                t.Amount,
			"state":                 t.State.String(),
			"closed":                !t.State.Open(),
			"registration_id":       chainjson.HexBytes(t.RegistrationID),
			"option":                t.Vote,
			"holding_account_id":    t.AccountID,
			"action_types":          actionTypes,
		})
	}
	return map[string]interface{}{
		"votes": votes,
		"last":  last,
	}, nil
}

type votingContractActionParams struct {
	TokenAssetID    *bc.AssetID        `json:"voting_token_asset_id,omitempty"`
	RightAssetID    *bc.AssetID        `json:"voting_right_asset_id,omitempty"`
	TxHash          *bc.Hash           `json:"transaction_id,omitempty"`
	TxIndex         *uint32            `json:"transaction_output,omitempty"`
	AccountID       string             `json:"account_id,omitempty"`        // right issuance, delegate, transfer, recall
	HolderScript    chainjson.HexBytes `json:"holder_script,omitempty"`     // right issuance, delegate, transfer
	AdminScript     chainjson.HexBytes `json:"admin_script,omitempty"`      // right, token issuance
	CanDelegate     *bool              `json:"can_delegate,omitempty"`      // delegate
	Amount          uint64             `json:"amount,omitempty"`            // token issuance
	Option          int64              `json:"option,omitempty"`            // vote
	RecallAccountID *string            `json:"recall_account_id,omitempty"` // override
	Delegates       []struct {
		HolderScript chainjson.HexBytes `json:"holder_script,omitempty"`
		AccountID    string             `json:"account_id,omitempty"`
	} `json:"override_delegates,omitempty"` // override
	Distributions []struct {
		RightAssetID *bc.AssetID `json:"voting_right_asset_id,omitempty"`
		Amount       uint64      `json:"amount,omitempty"`
	} `json:"distributions,omitempty"` // redistribute
	Registrations []struct {
		ID     chainjson.HexBytes `json:"id,omitempty"`
		Amount uint64             `json:"amount"`
	} `json:"registrations,omitempty"` // register
}

func (params *votingContractActionParams) token(ctx context.Context) (*voting.Token, error) {
	if params.TxHash == nil {
		return nil, errors.WithDetail(ErrBadBuildRequest, "missing voting token lot transaction_id")
	}
	if params.TxIndex == nil {
		return nil, errors.WithDetail(ErrBadBuildRequest, "missing voting token lot transaction_output")
	}
	token, err := voting.FindTokenForOutpoint(ctx, bc.Outpoint{Hash: *params.TxHash, Index: *params.TxIndex})
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.WithDetailf(ErrBadBuildRequest, "unknown voting token")
	}
	return token, err
}

func (params *votingContractActionParams) right(ctx context.Context) (*voting.Right, error) {
	if params.RightAssetID == nil {
		return nil, errors.WithDetail(ErrBadBuildRequest, "missing voting right asset id")
	}
	old, err := voting.GetCurrentHolder(ctx, *params.RightAssetID)
	if err == pg.ErrUserInputNotFound {
		return nil, errors.WithDetailf(ErrBadBuildRequest, "bad voting right source")
	}
	return old, err
}

func parseVotingAction(ctx context.Context, action *Action) (srcs []*txbuilder.Source, dsts []*txbuilder.Destination, err error) {
	var params votingContractActionParams
	err = action.UnmarshalInto(&params)
	if err != nil {
		return srcs, dsts, err
	}

	switch action.Type {
	case "issue-voting-right":
		if params.RightAssetID == nil {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "missing voting right asset id")
		}
		if params.AdminScript == nil {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting right issuance requires a voting system admin script")
		}
		holder, err := buildAddress(ctx, params.HolderScript, params.AccountID)
		if err != nil {
			return nil, nil, err
		}
		assetAmount := bc.AssetAmount{AssetID: *params.RightAssetID, Amount: 1}
		srcs = append(srcs, issuer.NewIssueSource(ctx, assetAmount, nil, nil))
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: assetAmount,
			Metadata:    action.Metadata,
			Receiver:    voting.RightIssuance(ctx, params.AdminScript, holder),
		})
	case "authenticate-voting-right":
		right, err := params.right(ctx)
		if err != nil {
			return nil, nil, err
		}
		reserver, receiver, err := voting.RightAuthentication(ctx, right)
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: right.AssetID, Amount: 1},
			Reserver:    reserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: right.AssetID, Amount: 1},
			Metadata:    action.Metadata,
			Receiver:    receiver,
		})
	case "recall-voting-right":
		if params.AccountID == "" {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "recall requires account ID to recall to")
		}
		old, err := params.right(ctx)
		if err != nil {
			return nil, nil, err
		}
		claims, err := voting.FindRightsForAsset(ctx, *params.RightAssetID)
		if err != nil {
			return nil, nil, err
		}
		if len(claims) < 2 {
			// You need at least two claims to have a recallable voting right.
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "bad voting right source")
		}

		// Find the earliest, active voting right claim that this account
		// has on this voting right token. We'll recall back to that point.
		var (
			recallPoint    *voting.Right
			recallPointIdx int
		)
		for i, claim := range claims {
			if claim.AccountID != nil && *claim.AccountID == params.AccountID {
				recallPoint = claim
				recallPointIdx = i
				break
			}
		}
		if recallPoint == nil {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting right not recallable")
		}
		reserver, receiver, err := voting.RightRecall(ctx, old, recallPoint, claims[recallPointIdx+1:len(claims)-1])
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: old.AssetID, Amount: 1},
			Reserver:    reserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: old.AssetID, Amount: 1},
			Metadata:    action.Metadata,
			Receiver:    receiver,
		})
	case "delegate-voting-right":
		old, err := params.right(ctx)
		if err != nil {
			return nil, nil, err
		}
		if !old.Delegatable {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "delegating this voting right is prohibited")
		}
		script, err := buildAddress(ctx, params.HolderScript, params.AccountID)
		if err != nil {
			return nil, nil, err
		}
		var delegatable = old.Delegatable
		if params.CanDelegate != nil {
			delegatable = *params.CanDelegate
		}
		reserver, receiver, err := voting.RightDelegation(ctx, old, script, delegatable)
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: old.AssetID, Amount: 1},
			Reserver:    reserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: old.AssetID, Amount: 1},
			Metadata:    action.Metadata,
			Receiver:    receiver,
		})
	case "transfer-voting-right":
		old, err := params.right(ctx)
		if err != nil {
			return nil, nil, err
		}
		script, err := buildAddress(ctx, params.HolderScript, params.AccountID)
		if err != nil {
			return nil, nil, err
		}
		reserver, receiver, err := voting.RightTransfer(ctx, old, script)
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: old.AssetID, Amount: 1},
			Reserver:    reserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: old.AssetID, Amount: 1},
			Metadata:    action.Metadata,
			Receiver:    receiver,
		})
	case "override-voting-right":
		if params.RightAssetID == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "missing voting right asset id")
		}
		// retrieve the entire current history
		history, err := voting.FindRightsForAsset(ctx, *params.RightAssetID)
		if err != nil {
			return nil, nil, err
		}
		if len(history) < 1 {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "cannot find voting right with asset id %s", *params.RightAssetID)
		}

		var (
			forkPoint          *voting.Right
			delegates          []voting.RightHolder
			intermediaryRights []*voting.Right
		)

		forkPoint = history[len(history)-1]
		// Find the recall point in the voting right history. Use that as the
		// fork point if provided.
		if params.RecallAccountID != nil {
			for idx, r := range history {
				if r.AccountID != nil && *r.AccountID == *params.RecallAccountID {
					forkPoint, intermediaryRights = r, history[idx+1:len(history)-1]
					break
				}
			}
			if forkPoint == history[len(history)-1] {
				return nil, nil, errors.WithDetail(ErrBadBuildRequest, "voting right not recallable")
			}
		}

		for _, d := range params.Delegates {
			script, err := buildAddress(ctx, d.HolderScript, d.AccountID)
			if err != nil {
				return nil, nil, err
			}

			rh := voting.RightHolder{
				Script: script,
			}
			delegates = append(delegates, rh)
		}
		reserver, receiver, err := voting.RightOverride(ctx, history[len(history)-1], forkPoint, intermediaryRights, delegates)
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: forkPoint.AssetID, Amount: 1},
			Reserver:    reserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: forkPoint.AssetID, Amount: 1},
			Metadata:    action.Metadata,
			Receiver:    receiver,
		})
	case "issue-voting-token":
		if params.RightAssetID == nil {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "new voting tokens must provide corresponding voting right")
		}
		if params.AdminScript == nil {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "new voting tokens must provide the voting system admin script")
		}

		assetAmount := bc.AssetAmount{AssetID: *params.TokenAssetID, Amount: params.Amount}
		srcs = append(srcs, issuer.NewIssueSource(ctx, assetAmount, nil, nil))
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: assetAmount,
			Metadata:    action.Metadata,
			Receiver:    voting.TokenIssuance(ctx, *params.RightAssetID, params.AdminScript),
		})
	case "redistribute-voting-token":
		token, err := params.token(ctx)
		if err != nil {
			return nil, nil, err
		}
		right, err := voting.GetCurrentHolder(ctx, token.Right)
		if err != nil {
			return nil, nil, err
		}
		if !token.State.Distributed() {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting token must be in distributed state to be redistributed")
		}
		if params.Distributions == nil {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "missing distribution mapping")
		}
		remaining := token.Amount
		distributions := make(map[bc.AssetID]uint64, len(params.Distributions))
		for _, d := range params.Distributions {
			if d.RightAssetID == nil {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "distribution without a voting right asset ID")
			}
			distributions[*d.RightAssetID] = d.Amount
			remaining = remaining - int64(d.Amount)
		}
		if remaining < 0 {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "distribution total exceeds voting token lot's amount")
		}

		tokenReserver, redistributeDsts, err := voting.TokenRedistribution(ctx, token, right.PKScript(), distributions)
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: token.AssetID, Amount: uint64(token.Amount)},
			Reserver:    tokenReserver,
		})
		dsts = append(dsts, redistributeDsts...)
	case "register-voting-token":
		token, err := params.token(ctx)
		if err != nil {
			return nil, nil, err
		}
		right, err := voting.GetCurrentHolder(ctx, token.Right)
		if err != nil {
			return nil, nil, err
		}
		if !token.State.Distributed() {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "voting token must be in distributed state")
		}
		if !token.State.Open() {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "voting has been closed")
		}
		registrations := make([]voting.Registration, 0, len(params.Registrations))
		change := token.Amount
		for _, r := range params.Registrations {
			registrations = append(registrations, voting.Registration{ID: r.ID, Amount: r.Amount})
			change = change - int64(r.Amount)
		}
		// Validate the registrations. The amounts should not exceed the lot amount.
		if change < 0 {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "voting token registration amounts exceed lot amount")
		}
		tokenReserver, registerDsts, err := voting.TokenRegistration(ctx, token, right.PKScript(), registrations)
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: token.AssetID, Amount: uint64(token.Amount)},
			Reserver:    tokenReserver,
		})
		dsts = append(dsts, registerDsts...)
	case "vote":
		token, err := params.token(ctx)
		if err != nil {
			return nil, nil, err
		}
		right, err := voting.GetCurrentHolder(ctx, token.Right)
		if err != nil {
			return nil, nil, err
		}
		if !token.State.Registered() && !token.State.Voted() {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting token must be in registered state")
		}
		if !token.State.Open() {
			return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting has been closed")
		}
		tokenReserver, tokenReceiver, err := voting.TokenVote(ctx, token, right.PKScript(), params.Option)
		if err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, &txbuilder.Source{
			AssetAmount: bc.AssetAmount{AssetID: token.AssetID, Amount: uint64(token.Amount)},
			Reserver:    tokenReserver,
		})
		dsts = append(dsts, &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: token.AssetID, Amount: uint64(token.Amount)},
			Receiver:    tokenReceiver,
		})
	case "close-vote":
		if params.TokenAssetID == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "missing voting token asset id")
		}
		votes, _, err := voting.GetVotes(ctx, []bc.AssetID{*params.TokenAssetID}, "", "", math.MaxInt64)
		if err != nil {
			return nil, nil, errors.Wrap(err, "finding voting tokens to close")
		}

		for _, v := range votes {
			if !v.State.Open() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting has already been closed")
			}
			reserver, receiver, err := voting.TokenFinish(ctx, v)
			if err != nil {
				return nil, nil, err
			}
			srcs = append(srcs, &txbuilder.Source{
				AssetAmount: bc.AssetAmount{AssetID: v.AssetID, Amount: uint64(v.Amount)},
				Reserver:    reserver,
			})
			dsts = append(dsts, &txbuilder.Destination{
				AssetAmount: bc.AssetAmount{AssetID: v.AssetID, Amount: uint64(v.Amount)},
				Receiver:    receiver,
			})
		}
	case "reset-voting-token":
		if params.TokenAssetID == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "missing voting token asset id")
		}
		votes, _, err := voting.GetVotes(ctx, []bc.AssetID{*params.TokenAssetID}, "", "", math.MaxInt64)
		if err != nil {
			return nil, nil, errors.Wrap(err, "finding voting tokens to reset")
		}

		for _, v := range votes {
			reserver, receiver, err := voting.TokenReset(ctx, v)
			if err != nil {
				return nil, nil, err
			}
			srcs = append(srcs, &txbuilder.Source{
				AssetAmount: bc.AssetAmount{AssetID: v.AssetID, Amount: uint64(v.Amount)},
				Reserver:    reserver,
			})
			dsts = append(dsts, &txbuilder.Destination{
				AssetAmount: bc.AssetAmount{AssetID: v.AssetID, Amount: uint64(v.Amount)},
				Metadata:    action.Metadata,
				Receiver:    receiver,
			})
		}
	case "retire-voting-token":
		if params.TokenAssetID == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "missing voting token asset id")
		}
		votes, _, err := voting.GetVotes(ctx, []bc.AssetID{*params.TokenAssetID}, "", "", math.MaxInt64)
		if err != nil {
			return nil, nil, errors.Wrap(err, "finding voting tokens to retire")
		}

		var totalAmount uint64
		for _, v := range votes {
			if v.State.Open() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting must be closed to retire tokens")
			}
			reserver, err := voting.TokenRetire(ctx, v)
			if err != nil {
				return nil, nil, err
			}
			srcs = append(srcs, &txbuilder.Source{
				AssetAmount: bc.AssetAmount{AssetID: v.AssetID, Amount: uint64(v.Amount)},
				Reserver:    reserver,
			})
			totalAmount += uint64(v.Amount)
		}
		dsts = append(dsts, txbuilder.NewRetireDestination(ctx, &bc.AssetAmount{AssetID: *params.TokenAssetID, Amount: totalAmount}, action.Metadata))
	case "invalidate-voting-token":
		if params.TokenAssetID == nil {
			return nil, nil, errors.WithDetail(ErrBadBuildRequest, "missing voting token asset id")
		}
		votes, _, err := voting.GetVotes(ctx, []bc.AssetID{*params.TokenAssetID}, "", "", math.MaxInt64)
		if err != nil {
			return nil, nil, errors.Wrap(err, "finding voting tokens to invalidate")
		}

		for _, v := range votes {
			if v.State.Open() {
				return nil, nil, errors.WithDetailf(ErrBadBuildRequest, "voting must be closed to retire tokens")
			}
			reserver, receiver, err := voting.TokenInvalidate(ctx, v)
			if err != nil {
				return nil, nil, err
			}
			srcs = append(srcs, &txbuilder.Source{
				AssetAmount: bc.AssetAmount{AssetID: v.AssetID, Amount: uint64(v.Amount)},
				Reserver:    reserver,
			})
			dsts = append(dsts, &txbuilder.Destination{
				AssetAmount: bc.AssetAmount{AssetID: v.AssetID, Amount: uint64(v.Amount)},
				Receiver:    receiver,
			})
		}
	default:
		err = errors.WithDetailf(ErrBadBuildRequest, "unknown voting action `%s`", action.Type)
	}
	return srcs, dsts, err
}
