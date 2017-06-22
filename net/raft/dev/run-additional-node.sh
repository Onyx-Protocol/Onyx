#!/bin/bash

if test $# -ne 1
then exec echo >&2 'Usage: run-additional-node.sh port'
fi

# Enables job control to allow backgrounding & foregrounding cored process
set -meo pipefail

port=$1
leader_address=localhost:1999
address=localhost:$port
db_url=postgres:///core0?sslmode=disable

mkdir -p $HOME/.chaincore$port/
cp $CHAIN_CORE_HOME/tls.* $HOME/.chaincore$port/

export CHAIN_CORE_HOME=$HOME/.chaincore$port

LISTEN=$address \
DATABASE_URL=$db_url \
cored &

CORE_URL=https://$address \
corectl join $leader_address

fg
