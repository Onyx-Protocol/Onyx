#!/bin/bash -e

# Delete the empty struct definitions of the
# placeholder Hash and AssetID types.
# This lets us define them ourselves in hash.go

sed -i '' '/type Hash struct {/ { N; d; }' bc.pb.go
sed -i '' '/type AssetID struct {/ { N; d; }' bc.pb.go
