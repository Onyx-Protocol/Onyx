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
newDatabase=$?

(
	/usr/bin/chain/cored
	echo 'cored' >$psmgr
) &

/usr/bin/chain/corectl wait

echo "Chain Core is online!"
echo "Listening on: http://localhost:1999"

if [[ $newDatabase -eq 0 ]]; then
  echo "Autogenerating acccess token with client-readwrite privileges..."
  /usr/bin/chain/corectl create-token client client-readwrite \
    | tee -a $initlog \
    | tail -n1 > /var/log/chain/client-token

  tail /var/log/chain/client-token | grep -q ":"
  if [[ $? -eq 0 ]]; then
    echo "Copy the whole line below:"
    tail /var/log/chain/client-token
  fi
fi

read exit_process <$psmgr
exit 1
