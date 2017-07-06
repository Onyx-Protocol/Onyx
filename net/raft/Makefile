# The following commands may be useful for developing and debugging
# our Raft packages.


build: build-lib
	go install -tags localhost_auth chain/cmd/cored
	go install chain/cmd/corectl
	dropdb --if-exists core0
	rm -rf $(HOME)/.chaincore?*/raft $(HOME)/.chaincore?*/rocks
	createdb core0

run0:
	dev/run-bootstrapped-node.sh

run1:
	dev/run-additional-node.sh 1998

run2:
	dev/run-additional-node.sh 1997

## the following is copied from the top-level makefile
## tktk tessr figure out if these should be consolidated
SNAPPY = $(CHAIN)/vendor/github.com/google/snappy
ROCKSDB = $(CHAIN)/vendor/github.com/facebook/rocksdb

## builds chain core with c dependencies, for development environments
build-dev: build-lib
	cd $(CHAIN) && go install -tags 'localhost_auth http_ok init_cluster' ./cmd/cored

## builds statically-linked c dependencies
build-lib:
	-mkdir $(SNAPPY)/build
	cd $(SNAPPY)/build && cmake ..
	cd $(SNAPPY)/build && $(MAKE)
	cd $(ROCKSDB) && $(MAKE) static_lib

## clean removes all data directories and c libraries
clean:
	rm -rf $(SNAPPY)/build
	rm -f $(ROCKSDB)/librocksdb.a
