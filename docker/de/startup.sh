#!/bin/sh

chown -R postgres /var/lib/postgresql/data
#TODO(ryandotsmith): ensure we deal with persistence and idempotency
su postgres -c 'initdb -D /var/lib/postgresql/data'
su postgres -c 'pg_ctl start -w -D /var/lib/postgresql/data -l /var/lib/postgresql/data/postgres.log'
su postgres -c 'createdb core' 
su postgres -c 'psql core -f /var/lib/chain/schema.sql'

/usr/bin/chain/cored
