//+build localhost_auth

package main

import (
	"chain/core"
	"chain/core/config"
	"chain/net/http/authz"
)

func init() {
	config.BuildConfig.LocalhostAuth = true
	for _, p := range core.Policies {
		builtinGrants = append(builtinGrants, &authz.Grant{
			Policy:    p,
			GuardType: "localhost",
		})
	}
}
