package core

import "testing"

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"https://hsm.chain.com":     "https://hsm.chain.com/",
		"https://hsm.chain.com:443": "https://hsm.chain.com/",
		"https://hsm.chain.com:80":  "https://hsm.chain.com:80/",
		"https://hsm.chain.com/":    "https://hsm.chain.com/",
		"https://hsm.chain.com:":    "https://hsm.chain.com/",
		"HTTPS://HSM.CHAIN.COM":     "https://hsm.chain.com/",
		"http://hsm.chain.com:80":   "http://hsm.chain.com/",
		"http://hsm.chain.com:443":  "http://hsm.chain.com:443/",
	}

	for u, want := range cases {
		t.Run(u, func(t *testing.T) {
			normalized, err := normalizeURL(u)
			if err != nil {
				t.Fatal(err)
			}
			if normalized.String() != want {
				t.Errorf("normalizeURL(%q) = %q, want %s", u, normalized.String(), want)
			}
		})
	}
}
