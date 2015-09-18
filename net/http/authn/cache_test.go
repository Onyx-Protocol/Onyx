package authn

import (
	"testing"
	"time"
)

func TestTokenCache(t *testing.T) {
	tokenCache := NewTokenCache()

	entries := []struct {
		id, secret, userID string
		exp                time.Time
	}{
		{"id1", "s1", "u1", time.Time{}},                // forever token
		{"id2", "s2", "u2", time.Now().Add(time.Hour)},  // unexpired token
		{"id3", "s3", "u3", time.Now().Add(-time.Hour)}, // expired token
	}
	for _, e := range entries {
		tokenCache.Store(e.id, e.secret, e.userID, e.exp)
	}

	cases := []struct {
		id, secret, want string
	}{
		{"id1", "s1", "u1"},
		{"id2", "s2", "u2"},
		{"id3", "s3", ""},
		{"id1", "badsecret", ""},
		{"id2", "badsecret", ""},
		{"id3", "badsecret", ""},
		{"badid", "s", ""},
	}

	for _, c := range cases {
		got := tokenCache.Get(c.id, c.secret)

		if got != c.want {
			t.Errorf("got Get(%q, %q) = %q want %q", c.id, c.secret, got, c.want)
		}
	}
}
