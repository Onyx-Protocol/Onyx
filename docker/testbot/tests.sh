#!/bin/bash

set -eou pipefail

initlog=/var/log/chain/init.log

# cleanup kills cored process
cleanup() {
	echo "Cleanup:" 1>&2
	for pid in $(ps aux | grep [c]ored$ | awk {'print $1'});
	do
		kill -9 $pid
	done
	wait
}
# call cleanup on program exit
trap 'cleanup' EXIT

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

echo "Running Basic Example"
/go/bin/corectl config-generator
/go/bin/cored 2>&1 | tee $initlog &
waitForGenerator
java -ea -cp /usr/bin/chain/chain-core-qa.jar com.chain_qa.BasicExample
