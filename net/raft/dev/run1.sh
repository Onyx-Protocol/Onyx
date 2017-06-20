#!/bin/bash

# Enables job control to allow backgrounding & foregrounding cored process
set -m


# TODO(vicki): add arguments to shell scripts so we can have one to run 
# all test nodes
export CHAIN_CORE_HOME=$HOME/.chaincore1

LISTEN=127.0.0.1:1998 \
DATABASE_URL=postgres:///core0?sslmode=disable \
cored &

CORE_URL=http://localhost:1998 \
corectl join 127.0.0.1:1999

fg
