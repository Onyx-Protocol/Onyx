#!/bin/bash

BUILD_PG_DIR="$PROJECT_DIR/pg/build"
TARGET_PG_DIR="$BUILT_PRODUCTS_DIR/$CONTENTS_FOLDER_PATH/Postgres"

if [ ! -e "${TARGET_PG_DIR}" ]
then
    # copy binaries
    cd "${BUILD_PG_DIR}/bin/"
    mkdir -p "${TARGET_PG_DIR}/bin/"
    # copy postgresql binaries
    cp clusterdb createdb createlang createuser dropdb droplang dropuser ecpg initdb pg* postgres postmaster psql reindexdb vacuumdb "${TARGET_PG_DIR}/bin/"

    # copy dynamic libraries only (no need for static libraries)
    cd "${BUILD_PG_DIR}/lib/"
    mkdir -p "${TARGET_PG_DIR}/lib/"
    cp -af *.dylib "${TARGET_PG_DIR}/lib/"
    cp -afR postgresql "${TARGET_PG_DIR}/lib/"

    # copy include, share
    rm -f "${TARGET_PG_DIR}/include/json"
    cp -afR "${BUILD_PG_DIR}/include" "${BUILD_PG_DIR}/share" "${TARGET_PG_DIR}/"
fi
