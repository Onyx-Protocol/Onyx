#!/bin/sh

chown -R postgres /var/lib/postgresql/data

su postgres -c 'initdb -D /var/lib/postgresql/data'
su postgres -c 'pg_ctl start -w -D /var/lib/postgresql/data -l /var/lib/postgresql/data/postgres.log'
su postgres -c 'createdb core' 
if [[ $? -eq 0 ]]; then
  su postgres -c 'psql core -f /var/lib/chain/schema.sql'
fi

/usr/bin/chain/corectl boot hello@chain.com password
/usr/bin/chain/cored &
/srv/chain/dashboard/bin/rails s --binding=0.0.0.0 -p 8081
