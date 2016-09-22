#!/bin/bash

set -eou pipefail

mkdir -p $GOPATH/log/testbot
initlog=$GOPATH/log/testbot/init.log

# cleanup kills cored process and removes built artifacts
#
# The function body is wrapped in a subshell so that failure
# of cleanup commands doesn't exit the script (due to set -e)
cleanup() {(
	set +e
	echo 'cleaning up' 1>&2
	for pid in $(ps aux | grep [c]ored$ | awk {'print $1'});
	do
		kill -9 $pid
	done
	wait
	rm -rf $CHAIN/sdk/java/target
	rm -rf $GOPATH/bin/cored
	rm -rf $GOPATH/bin/corectl
	dropdb core
)}
# call cleanup on program exit
trap 'cleanup' EXIT

# waitForGenerator blocks the script and greps
# the generator's output for a log message signifying
# the generator is fully initialized. It will timeout
# after 30s.
waitForGenerator() {(
	set +e
	start=`date +%s`
	while [ $(( `date +%s` - $start )) -lt 30 ]; do
		grep "I am the core leader" $initlog >/dev/null
		if [[ $? -eq 0 ]]; then
			break
		fi
	done
)}

echo 'setup database' 1>&2
createdb core
psql core -f $CHAIN/core/schema.sql
# TODO(boymanjor): generate credentials

echo 'installing cored' 1>&2
go install -tags 'insecure_disable_https_redirect' chain/cmd/cored

echo 'installing corectl' 1>&2
go install chain/cmd/corectl

echo 'running integration tests' 1>&2
$GOPATH/bin/corectl config-generator
LISTEN=":8081" $GOPATH/bin/cored 2>&1 | tee $initlog &
waitForGenerator
cd $CHAIN/sdk/java && mvn -Dchain.api.url="http://localhost:8081" integration-test
