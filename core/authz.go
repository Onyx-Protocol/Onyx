package core

const grantPrefix = "/core/grant/"

var policies = []string{
	"client-readwrite",
	"client-readonly",
	"crosscore-readonly",
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
	"/submit":                 {"client-readwrite"},

	networkRPCPrefix + "get-block":         {"crosscore-readonly"},
	networkRPCPrefix + "get-snapshot-info": {"crosscore-readonly"},
	networkRPCPrefix + "get-snapshot":      {"crosscore-readonly"},
	networkRPCPrefix + "signer/sign-block": {"crosscore-readonly"}, // TODO(tessr): make this crosscore-signblock
	networkRPCPrefix + "block-height":      {"crosscore-readonly"},

	"/list-authorization-grants":  {"client-readwrite", "client-readonly"},
	"/create-authorization-grant": {"client-readwrite"},
	"/delete-authorization-grant": {"client-readwrite"},
	"/create-access-token":        {"client-readwrite", "internal"},
	"/list-access-tokens":         {"client-readwrite", "client-readonly"},
	"/delete-access-token":        {"client-readwrite"},
	"/configure":                  {"client-readwrite", "internal"},
	"/info":                       {"client-readwrite", "client-readonly", "crosscore-readonly", "monitoring"},

	"/debug/vars":          {"client-readwrite", "client-readonly", "monitoring"}, // should monitoring endpoints also be available to any other policy-holders?
	"/debug/pprof":         {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/profile": {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/symbol":  {"client-readwrite", "client-readonly", "monitoring"},
	"/debug/pprof/trace":   {"client-readwrite", "client-readonly", "monitoring"},

	"/raft/join": {"internal"},
	"/raft/msg":  {"internal"},

	"/":                                       {"public"},
	"/dashboard":                              {"public"},
	"/dashboard/core":                         {"public"},
	"/dashboard/access_tokens/client":         {"public"},
	"/dashboard/access_tokens/client/create":  {"public"},
	"/dashboard/access_tokens/network":        {"public"},
	"/dashboard/access_tokens/network/create": {"public"},
	"/dashboard/transactions":                 {"public"},
	"/dashboard/transactions/create":          {"public"},
	"/dashboard/accounts":                     {"public"},
	"/dashboard/accounts/create":              {"public"},
	"/dashboard/assets":                       {"public"},
	"/dashboard/assets/create":                {"public"},
	"/dashboard/balances":                     {"public"},
	"/dashboard/unspents":                     {"public"},
	"/dashboard/mockhsms":                     {"public"},
	"/dashboard/transaction-feeds":            {"public"},
	"/dashboard/transaction-feeds/create":     {"public"},

	"/docs": {"public"},
}
