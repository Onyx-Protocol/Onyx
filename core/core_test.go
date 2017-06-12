package core

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"chain/core/config"
	"chain/core/leader"
	"chain/net"
	"chain/net/http/httpjson"
	"chain/testutil"
)

func TestForwardToLeader(t *testing.T) {
	// Create a test http server with TLS to be a fake leader process.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/info" {
			t.Fatalf("unexpected call to %s", req.URL.Path)
		}
		username, password, ok := req.BasicAuth()
		if ok && username != "" && password != "" {
			t.Error("request credentials shouldn't be forwarded")
		}
		rw.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(rw, `{
            "state": "leading",
            "is_configured": true
        }`)
	}))
	defer ts.Close()

	// TODO(jackson): In Go 1.9+ use ts.Client():
	// https://go-review.googlesource.com/c/34639/
	cert, err := x509.ParseCertificate(ts.TLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	certpool := x509.NewCertPool()
	certpool.AddCert(cert)

	// Setup a core.API so that it's a follower and leader.Address will
	// return the fake HTTPS server created above. Also, include its
	// certs in an internal httpClient so that it trusts the test server's
	// certs.
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	tlsConfig := net.DefaultTLSConfig()
	tlsConfig.RootCAs = certpool

	api := &API{
		config: &config.Config{},
		leader: alwaysFollower{leaderAddress: u.Host},
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}

	// Create a fake incoming request so that forwardToLeader can propagate
	// the basic auth credentials.
	fakeRequest, err := http.NewRequest("POST", "http://localhost:1999/info", nil)
	if err != nil {
		t.Fatal(err)
	}
	fakeRequest.SetBasicAuth("example", "password")
	ctx := context.Background()
	ctx = httpjson.WithRequest(ctx, fakeRequest)
	got, err := api.info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"state":         "leading",
		"is_configured": true,
	}
	if !testutil.DeepEqual(got, want) {
		t.Errorf("Got response %#v, want %#v", got, want)
	}
}

type alwaysFollower struct {
	leaderAddress string
}

func (af alwaysFollower) State() leader.ProcessState { return leader.Following }
func (af alwaysFollower) Address(context.Context) (string, error) {
	return af.leaderAddress, nil
}
