#!/bin/bash

SOURCE_DIR="$PROJECT_DIR"
TARGET_DIR="$BUILT_PRODUCTS_DIR/$CONTENTS_FOLDER_PATH/Resources"

export CHAIN="${SOURCE_DIR}/../.."
export GOPATH="${CHAIN}/../.."
export PATH="${PATH}:/usr/local/go/bin"

cd "${CHAIN}"
go install -tags insecure_disable_https_redirect ./cmd/cored
go install -tags insecure_disable_https_redirect ./cmd/corectl

cp -f "${GOPATH}/bin/cored" "${TARGET_DIR}/"
cp -f "${GOPATH}/bin/corectl" "${TARGET_DIR}/"

# `go build` is slower on repeated runs than `go install`
# go build -o "${TARGET_DIR}/cored"   -tags insecure_disable_https_redirect "./cmd/cored"
# go build -o "${TARGET_DIR}/corectl" -tags insecure_disable_https_redirect "./cmd/corectl"
