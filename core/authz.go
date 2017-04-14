package core

import (
	"github.com/golang/protobuf/proto"

	"chain/database/raft"
	"chain/errors"
	"chain/net/http/authz"
)

const grantPrefix = "/core/grant/"

var policies = []string{
	"client-readwrite",
	"client-readonly",
	"network",
	"monitoring",
	"internal",
}

var policyByRoute = map[string][]string{
	"/create-account":          []string{"client-readwrite"},
	"/create-asset":            []string{"client-readwrite"},
	"/update-account-tags":     []string{"client-readwrite"},
	"/build-transaction":       []string{"client-readwrite"},
	"/submit-transaction":      []string{"client-readwrite"},
	"/create-control-program":  []string{"client-readwrite"},
	"/create-account-receiver": []string{"client-readwrite"},
	"/create-transaction-feed": []string{"client-readwrite"},
	"/get-transaction-feed":    []string{"client-readwrite", "client-readonly"},
	"/update-transaction-feed": []string{"client-readwrite"},
	"/delete-transaction-feed": []string{"client-readwrite"},
	"/mockhsm":                 []string{}, // ???
	"/list-accounts":           []string{"client-readwrite", "client-readonly"},
	"/list-assets":             []string{"client-readwrite", "client-readonly"},
	"/list-transaction-feeds":  []string{"client-readwrite", "client-readonly"},
	"/list-transactions":       []string{"client-readwrite", "client-readonly"},
	"/list-balances":           []string{"client-readwrite", "client-readonly"},
	"/list-unspent-outputs":    []string{"client-readwrite", "client-readonly"},
	"/reset":                   []string{"client-readwrite"},
	"/submit":                  []string{"client-readwrite"},

	networkRPCPrefix + "get-block":         []string{"network"},
	networkRPCPrefix + "get-snapshot-info": []string{"network"},
	networkRPCPrefix + "get-snapshot":      []string{"network"},
	networkRPCPrefix + "signer/sign-block": []string{"network"},
	networkRPCPrefix + "block-height":      []string{"network"},

	"/list-acl-grants":     []string{"client-readwrite", "client-readonly"},
	"/create-acl-grant":    []string{"client-readwrite"},
	"/revoke-acl-grant":    []string{"client-readwrite"},
	"/create-access-token": []string{"client-readwrite"},
	"/list-access-tokens":  []string{"client-readwrite", "client-readonly"},
	"/delete-access-token": []string{"client-readwrite"},
	"/configure":           []string{"client-readwrite"},
	"/info":                []string{"client-readwrite", "client-readonly"}, // also network, maybe?

	"/debug/vars":          []string{"monitoring"}, // should monitoring endpoints also be available to any other policy-holders?
	"/debug/pprof":         []string{"monitoring"},
	"/debug/pprof/profile": []string{"monitoring"},
	"/debug/pprof/symbol":  []string{"monitoring"},
	"/debug/pprof/trace":   []string{"monitoring"},

	"/raft/join": []string{"internal"},
	"/raft/msg":  []string{"internal"},
}

func grantsByPolicies(raftDB *raft.Service, policies []string) ([]*authz.Grant, error) {
	var grants []*authz.Grant
	for _, p := range policies {
		data := raftDB.Stale().Get(grantPrefix + p)
		if data != nil {
			grantList := new(authz.GrantList)
			err := proto.Unmarshal(data, grantList)
			if err != nil {
				return nil, errors.Wrap(err)
			}
			grants = append(grants, grantList.GetGrants()...)
		}
	}
	return grants, nil
}
