#!/bin/bash

# Enables job control to allow backgrounding & foregrounding cored process
set -m

# Start initial node
LISTEN=127.0.0.1:1999 \
CHAIN_CORE_HOME=$HOME/.chaincore0 \
DATABASE_URL=postgres:///core0?sslmode=disable \
cored & 
corectl init
corectl allow-address 127.0.0.1:1998
corectl allow-address 127.0.0.1:1997
fg
