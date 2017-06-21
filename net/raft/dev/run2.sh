#!/bin/bash

# Enables job control to allow backgrounding & foregrounding cored process
set -m

# Move TLS certs to chaincore1
mkdir $HOME/.chaincore2
cp $CHAIN_CORE_HOME/tls.* $HOME/.chaincore2/

export CHAIN_CORE_HOME=$HOME/.chaincore2

LISTEN=localhost:1997 \
DATABASE_URL=postgres:///core0?sslmode=disable \
cored &

CORE_URL=https://localhost:1997 \
corectl join localhost:1999
fg
