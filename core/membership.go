package core

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"chain/database/sinkdb"
	"chain/errors"
	"chain/net/http/authz"
)

var (
	errMissingAddr = errors.New("missing address")
	errInvalidAddr = errors.New("invalid address")
)

func (a *API) addAllowedMember(ctx context.Context, x struct{ Addr string }) error {
	if x.Addr == "" {
		return errMissingAddr
	}
	hostname, _, err := net.SplitHostPort(x.Addr)
	if err != nil {
		newerr := errors.Sub(errInvalidAddr, err)
		if addrErr, ok := err.(*net.AddrError); ok {
			newerr = errors.WithDetail(newerr, addrErr.Err)
		}
		return newerr
	}

	// TODO(kr): create this and the below grant together atomically
	err = a.sdb.Exec(ctx, sinkdb.AddAllowedMember(x.Addr))
	if err != nil {
		return errors.Wrap(err)
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

	_, err = a.grants.Save(ctx, &authz.Grant{
		Policy:    "internal",
		GuardType: "x509",
		GuardData: guardData,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Protected: true,
	})
	return errors.Wrap(err)
}
