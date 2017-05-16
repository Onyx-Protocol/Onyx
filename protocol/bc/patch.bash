#!/bin/bash -e

# Delete the empty struct definitions of the
# placeholder Hash and AssetID types.
# This lets us define them ourselves in hash.go

sed -e '/type Hash struct {/ { N; d; }' bc.pb.go >bc.pb.go1
mv bc.pb.go1 bc.pb.go
sed -e '/type AssetID struct {/ { N; d; }' bc.pb.go >bc.pb.go1
mv bc.pb.go1 bc.pb.go
gofmt -w bc.pb.go
