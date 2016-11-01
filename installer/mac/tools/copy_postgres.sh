#!/bin/bash

SOURCE_DIR="$PROJECT_DIR/pg/build"
TARGET_DIR="$BUILT_PRODUCTS_DIR/$CONTENTS_FOLDER_PATH/Postgres"

cd "$SOURCE_DIR"

if [ ! -e "${TARGET_DIR}" ]
then
    # copy binaries
    cd "${SOURCE_DIR}/bin/"
    mkdir -p "${TARGET_DIR}/bin/"
    # copy postgresql binaries
    cp clusterdb createdb createlang createuser dropdb droplang dropuser ecpg initdb pg* postgres postmaster psql reindexdb vacuumdb "${TARGET_DIR}/bin/"

    # copy dynamic libraries only (no need for static libraries)
    cd "${SOURCE_DIR}/lib/"
    mkdir -p "${TARGET_DIR}/lib/"
    cp -af *.dylib "${TARGET_DIR}/lib/"
    cp -afR postgresql "${TARGET_DIR}/lib/"

    # copy include, share
    rm -f "${TARGET_DIR}/include/json"
    cp -afR "${SOURCE_DIR}/include" "${SOURCE_DIR}/share" "${TARGET_DIR}/"
fi
