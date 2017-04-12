package grant

// Generate code for the Grant and GrantList types.
//go:generate protoc -I. -I$CHAIN/.. --go_out=. grant.proto
