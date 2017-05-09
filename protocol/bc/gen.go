package bc

// Note that we discard the generated hash.pb.go.
// See the comment in hash.proto.

//go:generate protoc --go_out=. bc.proto hash.proto
//go:generate rm hash.pb.go
