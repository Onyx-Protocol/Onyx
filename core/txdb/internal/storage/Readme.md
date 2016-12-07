If you edit `snapshot.proto` (which specifies the serialization format for `state.Snapshot` objects) you will have to regenerate `snapshot.pb.go` using [protoc](https://github.com/google/protobuf#protocol-compiler-installation):

`protoc --go_out=. snapshot.proto`

You will also need [the `protoc` plugin for generating Go code](https://github.com/golang/protobuf/tree/master/protoc-gen-go).
