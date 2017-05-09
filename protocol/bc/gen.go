package bc

// Note that we do not generate hash.pb.go.
// See the comment in hash.proto.

//go:generate protoc -I. -I$CHAIN/.. --go_out=. bc.proto
