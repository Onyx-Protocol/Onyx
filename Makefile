## builds chain core with c dependencies, for development environments
build-dev: build-lib
	go install -tags 'localhost_auth http_ok reset' chain/cmd/cored

## builds statically-linked c dependencies
build-lib:
	cd $(CHAIN)/vendor/github.com/google/snappy; mkdir build; cd build && cmake ../ && make
	cd $(CHAIN)/vendor/github.com/facebook/rocksdb; make static_lib

## clean removes all data directories and c libraries
clean:
	rm -rf $(CHAIN)/vendor/github.com/google/snappy/build
	rm $(CHAIN)/vendor/github.com/facebook/rocksdb/librocksdb.a