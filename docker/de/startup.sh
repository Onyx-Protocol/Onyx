#!/bin/sh

psmgr=/tmp/chain-psmgr
rm -f $psmgr
mkfifo $psmgr

chown -R postgres /var/lib/postgresql/data
su postgres -c 'initdb -D /var/lib/postgresql/data'
su postgres -c 'pg_ctl start -w -D /var/lib/postgresql/data -l /var/lib/postgresql/data/postgres.log'
su postgres -c 'createdb core'
# TODO(kr): if createdb call is successful,
# generate credentials, print to stdout, and
# save to /var/log/chain/credentials.json

(
	/usr/bin/chain/cored
	echo 'cored' >$psmgr
) &

read exit_process <$psmgr
exit 1
