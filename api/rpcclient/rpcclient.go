package rpcclient

import (
	"errors"

	"chain/fedchain"
)

var (
	fc           *fedchain.FC
	generatorURL string
)

// ErrNoGenerator is returned by GetBlocks() when no remote generator
// has been configured.
var ErrNoGenerator = errors.New("no remote generator configured")

// Init initializes the client package.
func Init(chain *fedchain.FC, remoteGeneratorURL string) {
	fc = chain
	generatorURL = remoteGeneratorURL
}
