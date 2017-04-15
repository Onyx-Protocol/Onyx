package core

const grantPrefix = "/core/grant/"

var policies = []string{
	"client-readwrite",
	"client-readonly",
	"network",
	"monitoring",
	"internal",
}

var policyByRoute = map[string][]string{
	"/create-account":          {"client-readwrite"},
	"/create-asset":            {"client-readwrite"},
	"/update-account-tags":     {"client-readwrite"},
	"/build-transaction":       {"client-readwrite"},
	"/submit-transaction":      {"client-readwrite"},
	"/create-control-program":  {"client-readwrite"},
	"/create-account-receiver": {"client-readwrite"},
	"/create-transaction-feed": {"client-readwrite"},
	"/get-transaction-feed":    {"client-readwrite", "client-readonly"},
	"/update-transaction-feed": {"client-readwrite"},
	"/delete-transaction-feed": {"client-readwrite"},
	"/mockhsm":                 {"client-readwrite"},
	"/mockhsm/create-key":      {"client-readwrite"},
	"/mockhsm/list-keys":       {"client-readwrite", "client-readonly"},
	"/mockhsm/delkey":          {"client-readwrite"},

	"/list-accounts":          {"client-readwrite", "client-readonly"},
	"/list-assets":            {"client-readwrite", "client-readonly"},
	"/list-transaction-feeds": {"client-readwrite", "client-readonly"},
	"/list-transactions":      {"client-readwrite", "client-readonly"},
	"/list-balances":          {"client-readwrite", "client-readonly"},
	"/list-unspent-outputs":   {"client-readwrite", "client-readonly"},
	"/reset":                  {"client-readwrite"},
	"/submit":                 {"client-readwrite"},

	networkRPCPrefix + "get-block":         {"network"},
	networkRPCPrefix + "get-snapshot-info": {"network"},
	networkRPCPrefix + "get-snapshot":      {"network"},
	networkRPCPrefix + "signer/sign-block": {"network"},
	networkRPCPrefix + "block-height":      {"network"},

	"/list-acl-grants":     {"client-readwrite", "client-readonly"},
	"/create-acl-grant":    {"client-readwrite"},
	"/revoke-acl-grant":    {"client-readwrite"},
	"/create-access-token": {"client-readwrite"},
	"/list-access-tokens":  {"client-readwrite", "client-readonly"},
	"/delete-access-token": {"client-readwrite"},
	"/configure":           {"client-readwrite"},
	"/info":                {"client-readwrite", "client-readonly", "network", "monitoring"},

	"/debug/vars":          {"client-readwrite", "client-readonly", "monitoring"}, // should monitoring endpoints also be available to any other policy-holders?
	"/debug/pprof":         {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/profile": {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/symbol":  {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/trace":   {"client-readwrite", "client-readonly", "monitoring"},

	"/raft/join": {"internal"},
	"/raft/msg":  {"internal"},
}
