package blocksigner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chain/core/rpc"
	"chain/crypto/ed25519"
	"chain/protocol/bc/legacy"
)

func TestEnclaveClient(t *testing.T) {
	fakeSignature := []byte(`fakesignature`)

	// Create two test servers. One that always times out, the other
	// that will immediately return a correct signature.
	sv1 := httptest.NewServer(timeoutHandler())
	defer sv1.Close()
	sv2 := httptest.NewServer(fakeEnclavedHandler(t, "access-token-2", fakeSignature))
	defer sv2.Close()

	c := EnclaveClient{
		URLs: func() [][]string {
			return [][]string{
				{sv1.URL, "chain-core:access-token-1"},
				{sv2.URL, "chain-core:access-token-2"},
			}
		},
		BaseClient: rpc.Client{},
	}

	ctx := context.Background()
	var bh legacy.BlockHeader
	sig, err := c.Sign(ctx, make(ed25519.PublicKey, ed25519.PublicKeySize), &bh)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sig, fakeSignature) {
		t.Errorf("got signature %x, want %x", sig, fakeSignature)
	}
}

func timeoutHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		rw.WriteHeader(http.StatusRequestTimeout)
		json.NewEncoder(rw).Encode("Request timed out.")
	})
}

func fakeEnclavedHandler(t testing.TB, accessToken string, sig []byte) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, pass, ok := req.BasicAuth()
		if !ok || pass != accessToken {
			t.Fatalf("Got basic auth access token %q, want %q", pass, accessToken)
		}

		// Encode the signature as a json string containing the
		// base64-encoded bytes. This is how enclaved formats its response.
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(sig)
	})
}
