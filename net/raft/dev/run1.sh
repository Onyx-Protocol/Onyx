#!/bin/bash

# Enables job control to allow backgrounding & foregrounding cored process
set -m

# Move TLS certs to chaincore1
mkdir $HOME/.chaincore1
cp $CHAIN_CORE_HOME/tls.* $HOME/.chaincore1/

# TODO(vicki): add arguments to shell scripts so we can have one to run 
# all test nodes
export CHAIN_CORE_HOME=$HOME/.chaincore1

LISTEN=localhost:1998 \
DATABASE_URL=postgres:///core0?sslmode=disable \
cored &

CORE_URL=https://localhost:1998 \
corectl join localhost:1999

fg
