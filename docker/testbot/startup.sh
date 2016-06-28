#!/bin/sh

chown -R postgres $PGDATA
su postgres -c 'initdb -D $PGDATA'
su postgres -c 'pg_ctl start -w -l $PGDATA/postgres.log'

su postgres -c 'createdb core'
su postgres -c 'psql core -f /var/lib/chain/schema.sql'
/usr/bin/chain/corectl boot hello@chain.com password | tee /var/log/chain/credentials.json
export TOKEN_ID=`cat /var/log/chain/credentials.json | grep tokenID | tr -d ',"' | awk {'print $2'}`
export TOKEN_SECRET=`cat /var/log/chain/credentials.json | grep tokenSecret | tr -d ',"' | awk {'print $2'}`
export CORE1_AUTH_URL="http://$TOKEN_ID:$TOKEN_SECRET@localhost:8080"

su postgres -c 'createdb core2'
su postgres -c 'psql core2 -f /var/lib/chain/schema.sql'
DB_URL=$DB2_URL /usr/bin/chain/corectl boot hello@chain.com password | tee /var/log/chain/credentials.json
export TOKEN_ID=`cat /var/log/chain/credentials.json | grep tokenID | tr -d ',"' | awk {'print $2'}`
export TOKEN_SECRET=`cat /var/log/chain/credentials.json | grep tokenSecret | tr -d ',"' | awk {'print $2'}`
export CORE2_AUTH_URL="http://$TOKEN_ID:$TOKEN_SECRET@localhost:8081"

su postgres -c 'createdb core3'
su postgres -c 'psql core3 -f /var/lib/chain/schema.sql'
DB_URL=$DB3_URL /usr/bin/chain/corectl boot hello@chain.com password | tee /var/log/chain/credentials.json
export TOKEN_ID=`cat /var/log/chain/credentials.json | grep tokenID | tr -d ',"' | awk {'print $2'}`
export TOKEN_SECRET=`cat /var/log/chain/credentials.json | grep tokenSecret | tr -d ',"' | awk {'print $2'}`
export CORE3_AUTH_URL="http://$TOKEN_ID:$TOKEN_SECRET@localhost:8082"

sh /usr/bin/chain/tests.sh
