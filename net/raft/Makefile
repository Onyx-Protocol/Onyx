# The following commands may be useful for developing and debugging
# our Raft packages. 


build:
	go install -tags localhost_auth chain/cmd/cored
	go install chain/cmd/corectl
	dropdb --if-exists core0
	rm -rf $(HOME)/.chaincore?*/raft
	createdb core0

run0:
	dev/run-bootstrapped-node.sh

run1:
	dev/run-additional-node.sh 1998

run2:
	dev/run-additional-node.sh 1997
