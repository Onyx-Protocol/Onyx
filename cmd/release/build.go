package main

type product struct {
	name string // e.g. chain-core-server, sdk-java, chain-enclave
	prv  bool   // whether it's built from chainprv

	// Build builds and packages the release, leaving the
	// results in one or more files on disk.
	// It returns a slice of file names for whatever it built.
	// e.g. chain-core-server-1.1-linux-amd64.tar.gz
	build func(p product, version, tagName string) ([]string, error)
}

var products = []product{
	chainCoreServer,
}
