// Package release reads release definitions
// from the Chain Core code repository.
package release

import (
	"fmt"
	"sort"
	"sync"

	"chain/errors"
)

const (
	EnvVar = "CHAIN"
	Path   = "build/release.txt"
)

var (
	releases []*Definition
	loadErr  error
	once     sync.Once
)

// Check returns an error if there was a problem loading the release table.
func Check() error {
	once.Do(load)
	return loadErr
}

// Definition describes a release.
// It contains enough information to construct (or reconstruct)
// that release, given a copy of the chain and chainprv repos.
type Definition struct {
	Product        string
	Version        string
	ChainCommit    string
	ChainprvCommit string
	Codename       string
}

// Get finds the given release in the release table and returns it.
// If version is the empty string, it returns the newest release
// for product.
func Get(product, version string) (*Definition, error) {
	once.Do(load)
	if loadErr != nil {
		return nil, loadErr
	}
	t := releases
	if version == "" {
		return getNewest(product, t)
	}
	i := sort.Search(len(t), func(i int) bool {
		return !less(t[i], &Definition{Product: product, Version: version})
	})
	if i >= len(t) || t[i].Product != product || t[i].Version != version {
		return nil, errors.Wrap(fmt.Errorf("not found: %s %s", product, version))
	}
	return t[i], nil
}

// getNewest returns the newest release for product.
func getNewest(product string, t []*Definition) (*Definition, error) {
	i := sort.Search(len(t), func(i int) bool {
		return t[i].Product > product
	}) - 1
	if i < 0 || t[i].Product != product {
		return nil, errors.Wrap(errors.New("not found: " + product))
	}
	return t[i], nil
}

func less(a, b *Definition) bool {
	if a.Product != b.Product {
		return a.Product < b.Product
	}
	return Less(a.Version, b.Version)
}
