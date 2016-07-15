package rpcclient

import "errors"

var generatorURL string

// ErrNoGenerator is returned by GetBlocks() when no remote generator
// has been configured.
var ErrNoGenerator = errors.New("no remote generator configured")

// Init initializes the client package.
func Init(remoteGeneratorURL string) {
	generatorURL = remoteGeneratorURL
}
