package authn

import (
	"crypto/subtle"
	"sync"
	"time"
)

// TokenCache provides a mechanism for avoiding repeated work
// authenticating username/password tokens. Each cached
// item can have an expiration, at which point the item
// is invalid, and the authentication process must be followed
// through again.
type TokenCache struct {
	sync.Mutex // protects the folllowing
	tokens     map[string]cacheItem
}

type cacheItem struct {
	secret []byte
	exp    time.Time
	uid    string
}

// NewTokenCache creates a new TokenCache
func NewTokenCache() *TokenCache {
	return &TokenCache{tokens: make(map[string]cacheItem)}
}

// Get returns the user ID for authentication parameters
// if it has been stored in the cache, and has not expired.
func (tc *TokenCache) Get(id, secret string) string {
	tc.Lock()
	item, ok := tc.tokens[id]
	tc.Unlock()
	if ok {
		expired := !item.exp.IsZero() && time.Now().After(item.exp)
		valid := subtle.ConstantTimeCompare(item.secret, []byte(secret)) == 1
		if valid && !expired {
			return item.uid
		}
	}
	return ""
}

// Store adds valid authentication parameters to the cache
// along with an expiration and userID.
func (tc *TokenCache) Store(id, secret, userID string, expiration time.Time) {
	tc.Lock()
	defer tc.Unlock()
	tc.tokens[id] = cacheItem{[]byte(secret), expiration, userID}
}
