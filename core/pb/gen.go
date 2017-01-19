package pb

//go:generate protoc --go_out=plugins=grpc:. -I $CHAIN/protobufs $CHAIN/protobufs/chaincore.proto
