#!/bin/bash

set -eou pipefail

initlog=/var/log/chain/init.log
export CHAIN_API_URL=$CORE1_AUTH_URL
export SECOND_API_URL=$CORE2_AUTH_URL

# cleanup kills cored processes and resets their blockchains
cleanup() {
	echo "Cleanup:" 1>&2
	for pid in $(ps aux | grep [c]ored$ | awk {'print $1'});
	do
		kill -9 $pid
	done
	wait
	cat /var/lib/chain/clear-blockchain.sql | psql $DB1_URL
	cat /var/lib/chain/clear-blockchain.sql | psql $DB2_URL
	cat /var/lib/chain/clear-blockchain.sql | psql $DB3_URL
}
# call cleanup on program exit
trap 'cleanup' EXIT

runTests() {
	# run multi-core tests
	java -ea -cp /usr/bin/chain/chain-core-qa.jar chain.qa.baseline.multicore.Main
}

# waitForGenerator blocks the script and greps
# the generator's output for a log message signifying
# the generator is fully initialized. It will timeout
# after 30s.
#
# The function body is wrapped in a subshell so that failure
# of the grep command doesn't exit the script (due to set -e)
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

echo "Config: 1 generator+signer"
/usr/bin/chain/cored 2>&1 | tee $initlog &

waitForGenerator
java -ea -cp /usr/bin/chain/chain-core-qa.jar chain.qa.baseline.singlecore.Main
cleanup

echo "Config: 1-of-1 sigs required; 1 generator, 1 remote signer"
GENERATOR=0 REMOTE_GENERATOR_URL=$CORE1_URL LISTEN=$CORE2_LISTEN \
	BLOCK_KEY=$PRIV1 DB_URL=$DB2_URL /usr/bin/chain/cored &

SIGNER=0 REMOTE_SIGNER_URLS=$CORE2_URL REMOTE_SIGNER_KEYS=$PUB1 \
	/usr/bin/chain/cored 2>&1 | tee $initlog &

waitForGenerator
runTests
cleanup

echo "Config: 1-of-2 sigs required; 1 generator+signer, 1 remote signer"
GENERATOR=0 REMOTE_GENERATOR_URL=$CORE1_URL LISTEN=$CORE2_LISTEN \
	BLOCK_KEY=$PRIV1 DB_URL=$DB2_URL /usr/bin/chain/cored &

REMOTE_SIGNER_URLS=$CORE2_URL REMOTE_SIGNER_KEYS=$PUB1 \
	/usr/bin/chain/cored 2>&1 | tee $initlog &

waitForGenerator
runTests
cleanup

echo "Config: 1-of-2 sigs required; 1 generator, 2 remote signers"
GENERATOR=0 REMOTE_GENERATOR_URL=$CORE1_URL LISTEN=$CORE2_LISTEN \
	BLOCK_KEY=$PRIV1 DB_URL=$DB2_URL /usr/bin/chain/cored &

GENERATOR=0 REMOTE_GENERATOR_URL=$CORE1_URL LISTEN=$CORE3_LISTEN \
	BLOCK_KEY=$PRIV2 DB_URL=$DB3_URL /usr/bin/chain/cored &

SIGNER=0 REMOTE_SIGNER_URLS=$CORE2_URL,$CORE3_URL \
	REMOTE_SIGNER_KEYS=$PUB1,$PUB2 /usr/bin/chain/cored 2>&1 | tee $initlog &

waitForGenerator
runTests
cleanup

echo "Config: 2-of-2 sigs required; 1 generator+signer, 1 signer"
GENERATOR=0 REMOTE_GENERATOR_URL=$CORE1_URL LISTEN=$CORE2_LISTEN \
	BLOCK_KEY=$PRIV1 DB_URL=$DB2_URL /usr/bin/chain/cored &

REMOTE_SIGNER_URLS=$CORE2_URL REMOTE_SIGNER_KEYS=$PUB1 SIGS_REQUIRED=2 \
	/usr/bin/chain/cored 2>&1 | tee $initlog &

waitForGenerator
runTests
cleanup

echo "Config: 2-of-2 sigs required; 1 generator, 2 remote signers"
GENERATOR=0 REMOTE_GENERATOR_URL=$CORE1_URL LISTEN=$CORE2_LISTEN \
	BLOCK_KEY=$PRIV1 DB_URL=$DB2_URL /usr/bin/chain/cored &

GENERATOR=0 REMOTE_GENERATOR_URL=$CORE1_URL LISTEN=$CORE3_LISTEN \
	BLOCK_KEY=$PRIV2 DB_URL=$DB3_URL /usr/bin/chain/cored &

SIGNER=0 REMOTE_SIGNER_URLS=$CORE2_URL,$CORE3_URL REMOTE_SIGNER_KEYS=$PUB1,$PUB2 \
	SIGS_REQUIRED=2 /usr/bin/chain/cored 2>&1 | tee $initlog &

waitForGenerator
runTests
