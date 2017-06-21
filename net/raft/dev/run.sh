#!/bin/bash

# Enables job control to allow backgrounding & foregrounding cored process
set -m

db_url=postgres:///core0?sslmode=disable
addresses=(
	'127.0.0.1:1999'
	'127.0.0.1:1998'
	'127.0.0.1:1997'
)

node=$1

mkdir $HOME/.chaincore$node/
mv $CHAIN_CORE_HOME/tls* $HOME/.chaincore$node/

LISTEN='${addresses[node]}' \
CHAIN_CORE_HOME=$HOME/.chaincore$node \
DATABASE_URL=$db_url \
cored &

if [ $node -eq 0 ]
then
	corectl init
	corectl allow-address ${addresses[1]}
	corectl allow-address ${addresses[2]}
else 
	CORE_URL=${addresses[node]}
	corectl join ${addresses[0]}
fi


fg
