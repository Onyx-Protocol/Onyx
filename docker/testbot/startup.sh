#!/bin/sh
set -eu

# setup postgres
mkdir -p ~/postgresql/data
initdb -D ~/postgresql/data
pg_ctl start -D ~/postgresql/data -w -l ~/postgresql/data/postgres.log

# setup and run testbot
echo "machine github.com login chainbot password $GITHUB_TOKEN" >> ~/.netrc
git clone https://github.com/chain/chain.git $CHAIN
cd $CHAIN/sdk/java && mvn package && rm -rf $CHAIN/sdk/java/target
/usr/local/go/bin/go install chain/cmd/testbot
exec $GOPATH/bin/testbot
