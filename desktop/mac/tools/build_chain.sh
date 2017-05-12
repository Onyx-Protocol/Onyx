#!/bin/bash

SOURCE_DIR="$PROJECT_DIR"
TARGET_DIR="$BUILT_PRODUCTS_DIR/$CONTENTS_FOLDER_PATH/Resources"

export CHAIN="${SOURCE_DIR}/../.."
export GOPATH="${CHAIN}/../.."
export PATH="${PATH}:/usr/local/go/bin"

tempBuildPath=`mktemp -d`
trap "rm -rf $tempBuildPath" EXIT
"${CHAIN}/bin/build-cored-release" chain-core-server-1.2.0 $tempBuildPath

cp -f $tempBuildPath/cored "${TARGET_DIR}/"
cp -f $tempBuildPath/corectl "${TARGET_DIR}/"
