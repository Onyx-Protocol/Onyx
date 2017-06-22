#!/bin/bash

# Enables job control to allow backgrounding & foregrounding cored process
set -meo pipefail

# Move TLS certs to chaincore0
mkdir -p $HOME/.chaincore0
cp $CHAIN_CORE_HOME/tls.* $HOME/.chaincore0/

# Start initial node
LISTEN=localhost:1999 \
CHAIN_CORE_HOME=$HOME/.chaincore0 \
DATABASE_URL=postgres:///core0?sslmode=disable \
cored &

corectl init
corectl allow-address localhost:1998
corectl allow-address localhost:1997
fg
