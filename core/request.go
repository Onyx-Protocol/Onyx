package core

import (
	"context"

	"chain/core/pb"
	"chain/errors"
)

var (
	errBadActionType = errors.New("bad action type")
	errBadAlias      = errors.New("bad alias")
	errBadAction     = errors.New("bad action object")
)

func (h *Handler) filterAliases(ctx context.Context, br *pb.BuildTxsRequest_Request) error {
	for i, a := range br.Actions {
		var err error
		switch a.Action.(type) {
		case *pb.Action_ControlAccount_:
			err = h.filterAssetAlias(ctx, i, a.GetControlAccount().Asset)
			if err == nil {
				err = h.filterAccountAlias(ctx, i, a.GetControlAccount().Account)
			}
		case *pb.Action_SpendAccount_:
			err = h.filterAssetAlias(ctx, i, a.GetSpendAccount().Asset)
			if err == nil {
				err = h.filterAccountAlias(ctx, i, a.GetSpendAccount().Account)
			}
		case *pb.Action_ControlProgram_:
			err = h.filterAssetAlias(ctx, i, a.GetControlProgram().Asset)
		case *pb.Action_Issue_:
			err = h.filterAssetAlias(ctx, i, a.GetIssue().Asset)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) filterAssetAlias(ctx context.Context, i int, asset *pb.AssetIdentifier) error {
	switch asset.Identifier.(type) {
	case *pb.AssetIdentifier_AssetAlias:
		a, err := h.Assets.FindByAlias(ctx, asset.GetAssetAlias())
		if err != nil {
			return errors.WithDetailf(err, "invalid asset alias %s on action %d", asset.GetAssetAlias(), i)
		}
		asset.SetAssetId(a.AssetID[:])
	}
	return nil
}

func (h *Handler) filterAccountAlias(ctx context.Context, i int, account *pb.AccountIdentifier) error {
	switch account.Identifier.(type) {
	case *pb.AccountIdentifier_AccountAlias:
		a, err := h.Accounts.FindByAlias(ctx, account.GetAccountAlias())
		if err != nil {
			return errors.WithDetailf(err, "invalid account alias %s on action %d", account.GetAccountAlias(), i)
		}
		account.SetAccountId(a.ID)
	}
	return nil
}
