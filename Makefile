SNAPPY = $(CHAIN)/vendor/github.com/google/snappy
ROCKSDB = $(CHAIN)/vendor/github.com/facebook/rocksdb
DB_DEV = core
RAFT_DEV = $(or $(CHAIN_CORE_HOME), $(HOME)/.chaincore)/raft

default: run

## run a development version of Chain Core at http://localhost:1999
run: build-dev
	cored

## reset the development database
resetdb:
	-dropdb $(DB_DEV)
	createdb $(DB_DEV)
	rm -rf $(RAFT_DEV)

## delete the development database, re-configure core, and run server
rerun: build-dev corectl resetdb
	sh -c "corectl wait; corectl config-generator" &
	cored

## run development dashboard at http://localhost:3000
dashserve:
	npm --prefix dashboard start

## run development documentation server at http://localhost:8080
docserve: md2html
	md2html serve

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

corectl:
	go install chain/cmd/corectl

md2html:
	go install chain/cmd/md2html
