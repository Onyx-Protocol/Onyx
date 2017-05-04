package core

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"chain/errors"
	"chain/net/http/authz"
)

var errMissingAddr = errors.New("missing address")

func (a *API) addAllowedMember(ctx context.Context, x struct{ Addr string }) error {
	if x.Addr == "" {
		return errMissingAddr
	}
	err := a.raftDB.AddAllowedMember(ctx, x.Addr)
	if err != nil {
		return errors.Wrap(err)
	}

	hostname, _, err := net.SplitHostPort(x.Addr)
	if err != nil {
		return errors.Wrap(err)
	}

	// only create a grant if we're using TLS
	if !a.useTLS {
		return nil
	}

	data := map[string]interface{}{
		"subject": map[string]string{
			"CN": hostname,
		},
	}

	guardData, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err)
	}

	grant := authz.Grant{
		Policy:    "internal",
		GuardType: "x509",
		GuardData: guardData,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Protected: true,
	}

	_, err = authz.StoreGrant(ctx, a.raftDB, grant, GrantPrefix)
	return errors.Wrap(err)
}
