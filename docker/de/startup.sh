#!/bin/sh

psmgr=/tmp/chain-psmgr
rm -f $psmgr
mkfifo $psmgr

echo "Initializing Chain Core..."
initlog=/var/log/chain/init.log
chown -R postgres /var/lib/postgresql/data
su postgres -c 'initdb -D /var/lib/postgresql/data' >$initlog 2>&1
su postgres -c 'pg_ctl start -w -D /var/lib/postgresql/data -l /var/lib/postgresql/data/postgres.log' >>$initlog 2>&1
su postgres -c 'createdb core' >>$initlog 2>&1
if [[ $? -eq 0 ]]; then
	/usr/bin/chain/corectl create-token client | tee -a $initlog | tail -n1 >/var/log/chain/client-token
fi
(
	/usr/bin/chain/cored
	echo 'cored' >$psmgr
) &
echo "Listening on: http://localhost:1999"
echo "Client access token: `tail /var/log/chain/client-token`"
echo "Chain Core is online!"
read exit_process <$psmgr
exit 1
