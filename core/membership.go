package core

import (
	"context"

	"chain/errors"
)

var errMissingAddr = errors.New("missing address")

func (a *API) addAllowedMember(ctx context.Context, x struct {
	Addr string `json:"addr"`
}) error {
	if x.Addr == "" {
		return errMissingAddr
	}
	return a.raftDB.AddAllowedMember(ctx, x.Addr)
	// TODO(tessr): create grant for this new member
}
