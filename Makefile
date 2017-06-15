SNAPPY = $(CHAIN)/vendor/github.com/google/snappy
ROCKSDB = $(CHAIN)/vendor/github.com/facebook/rocksdb

null:
	@echo Please specify a make target.

## builds chain core with c dependencies, for development environments
build-dev: build-lib
	go install -tags 'localhost_auth http_ok reset' chain/cmd/cored

## builds statically-linked c dependencies
build-lib:
	-mkdir $(SNAPPY)/build
	cd $(SNAPPY)/build && cmake ..
	cd $(SNAPPY)/build && $(MAKE)
	cd $(ROCKSDB); $(MAKE) static_lib

## clean removes all data directories and c libraries
clean:
	rm -rf $(SNAPPY)/build
	rm $(ROCKSDB)/librocksdb.a