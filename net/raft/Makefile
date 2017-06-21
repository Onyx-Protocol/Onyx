# The following commands may be useful for developing and debugging
# our Raft packages. 


build:
	go install -tags localhost_auth chain/cmd/cored
	go install chain/cmd/corectl
	dropdb --if-exists core0
	rm -rf $(HOME)/.chaincore{0,1,2}
	createdb core0

run0:
	dev/run0.sh

run1:
	dev/run1.sh

run2:
	dev/run2.sh
