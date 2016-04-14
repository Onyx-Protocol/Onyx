package rpcclient

import (
	"errors"

	"chain/cos"
)

var (
	fc           *cos.FC
	generatorURL string
)

// ErrNoGenerator is returned by GetBlocks() when no remote generator
// has been configured.
var ErrNoGenerator = errors.New("no remote generator configured")

// Init initializes the client package.
func Init(chain *cos.FC, remoteGeneratorURL string) {
	fc = chain
	generatorURL = remoteGeneratorURL
}
