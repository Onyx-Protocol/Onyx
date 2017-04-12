package core

const policyByRoute = map[string][]string{
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

	"/create-access-token": []string{"client-readwrite"},
	"/list-access-tokens":  []string{"client-readwrite", "client-readonly"},
	"/delete-access-token": []string{"client-readwrite"},
	"/configure":           []string{"client-readwrite"},
	"/info":                []string{"client-readwrite", "client-readonly"}, // also network, maybe?

	"/debug/vars":          []string{"monitoring"},
	"/debug/pprof":         []string{"monitoring"},
	"/debug/pprof/profile": []string{"monitoring"},
	"/debug/pprof/symbol":  []string{"monitoring"},
	"/debug/pprof/trace":   []string{"monitoring"},

	"/raft/*": []string{"internal"}, // tktk no regex probs
}
