package core

const grantPrefix = "/core/grant/"

var policies = []string{
	"client-readwrite",
	"client-readonly",
	"crosscore",
	"crosscore-signblock",
	"monitoring",
	"internal",
	"public",
}

var policyByRoute = map[string][]string{
	"/create-account":           {"client-readwrite"},
	"/create-asset":             {"client-readwrite"},
	"/update-account-tags":      {"client-readwrite"},
	"/update-asset-tags":        {"client-readwrite"},
	"/build-transaction":        {"client-readwrite"},
	"/submit-transaction":       {"client-readwrite"},
	"/create-control-program":   {"client-readwrite"},
	"/create-account-receiver":  {"client-readwrite"},
	"/create-transaction-feed":  {"client-readwrite"},
	"/get-transaction-feed":     {"client-readwrite", "client-readonly"},
	"/update-transaction-feed":  {"client-readwrite"},
	"/delete-transaction-feed":  {"client-readwrite"},
	"/mockhsm":                  {"client-readwrite"},
	"/mockhsm/create-block-key": {"internal"},
	"/mockhsm/create-key":       {"client-readwrite"},
	"/mockhsm/list-keys":        {"client-readwrite", "client-readonly"},
	"/mockhsm/delkey":           {"client-readwrite"},
	"/mockhsm/sign-transaction": {"client-readwrite"},

	"/list-accounts":          {"client-readwrite", "client-readonly"},
	"/list-assets":            {"client-readwrite", "client-readonly"},
	"/list-transaction-feeds": {"client-readwrite", "client-readonly"},
	"/list-transactions":      {"client-readwrite", "client-readonly"},
	"/list-balances":          {"client-readwrite", "client-readonly"},
	"/list-unspent-outputs":   {"client-readwrite", "client-readonly"},
	"/reset":                  {"client-readwrite", "internal"},

	crosscoreRPCPrefix + "submit":            {"crosscore", "crosscore-signblock"},
	crosscoreRPCPrefix + "get-block":         {"crosscore", "crosscore-signblock"},
	crosscoreRPCPrefix + "get-snapshot-info": {"crosscore", "crosscore-signblock"},
	crosscoreRPCPrefix + "get-snapshot":      {"crosscore", "crosscore-signblock"},
	crosscoreRPCPrefix + "signer/sign-block": {"crosscore-signblock"},
	crosscoreRPCPrefix + "block-height":      {"crosscore", "crosscore-signblock"},

	"/list-authorization-grants":  {"client-readwrite", "client-readonly", "internal"},
	"/create-authorization-grant": {"client-readwrite", "internal"},
	"/delete-authorization-grant": {"client-readwrite", "internal"},
	"/create-access-token":        {"client-readwrite", "internal"},
	"/list-access-tokens":         {"client-readwrite", "client-readonly"},
	"/delete-access-token":        {"client-readwrite"},
	"/configure":                  {"client-readwrite", "internal"},
	"/info":                       {"client-readwrite", "client-readonly", "crosscore", "crosscore-signblock", "monitoring"},

	"/debug/vars":          {"client-readwrite", "client-readonly", "monitoring"}, // should monitoring endpoints also be available to any other policy-holders?
	"/debug/pprof":         {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/profile": {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/symbol":  {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/trace":   {"client-readwrite", "client-readonly", "monitoring"},

	"/raft/join": {"internal"},
	"/raft/msg":  {"internal"},

	"/dashboard":  {"public"},
	"/dashboard/": {"public"},
}
