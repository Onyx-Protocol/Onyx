#!/bin/sh

chown -R postgres $PGDATA
su postgres -c 'initdb -D $PGDATA'
su postgres -c 'pg_ctl start -w -l $PGDATA/postgres.log'
su postgres -c 'createdb core'
su postgres -c 'psql core -f $CHAIN/core/schema.sql'
# TODO(boymanjor): generate credentials, print to stdout,
# and save to /var/log/chain/credentials.json

echo 'installing cored'
go install -tags 'insecure_disable_https_redirect' chain/cmd/cored

echo 'installing corectl'
go install chain/cmd/corectl

cd $CHAIN/sdk/java && mvn package
mvn org.apache.maven.plugins:maven-install-plugin:2.5.2:install-file -Dfile=target/chain-sdk-java-1.0.0.jar
cd $CHAIN/qa && mvn package
cp target/chain-core-qa.jar /usr/bin/chain/chain-core-qa.jar
sh $CHAIN/docker/testbot/tests.sh
