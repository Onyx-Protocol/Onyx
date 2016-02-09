package rpc

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"

	"chain/net/http/authn"
	"chain/net/http/reqid"
)

// NodeInfo encapsulates metadata about a node.
type NodeInfo struct {
	ProcessID string
	Target    string
	BuildTag  string
}

func (ni NodeInfo) String() string {
	return fmt.Sprintf("Chain; target=%s; process=%s; buildtag=%s",
		ni.Target, ni.ProcessID, ni.BuildTag)
}

func (ni NodeInfo) MarshalText() ([]byte, error) {
	return []byte(ni.String()), nil
}

var (
	// LocalNode includes data about this node. This information is used in
	// RPCs to identify the the node performing the RPC. This should be
	// initialized prior to using this package to avoid synchronization
	// issues.
	LocalNode NodeInfo
)

// SecretToken is a shared secret used by nodes (manager, issuer, generator,
// signer, etc) to authorize communication with each other. It must be set
// before calling any functions in this package.
var SecretToken string

// Authenticate compares the secret to the SecretToken used to authenticate
// requests. The api package uses this function to auth internode requests.
// id and ctx are unused because this is a chain/net/http/authn AuthFunc.
func Authenticate(ctx context.Context, id, secret string) (userID string, err error) {
	if subtle.ConstantTimeCompare([]byte(secret), []byte(SecretToken)) == 0 {
		return "", authn.ErrNotAuthenticated
	}

	return "", nil
}

// ErrStatusCode is an error returned when an rpc fails with a non-200
// response code.
type ErrStatusCode struct {
	URL        string
	StatusCode int
}

func (e ErrStatusCode) Error() string {
	return fmt.Sprintf("Request to `%s` responded with %d %s",
		e.URL, e.StatusCode, http.StatusText(e.StatusCode))
}

// Call calls a remote procedure on another node, specified by the path.
func Call(ctx context.Context, address, path string, request, response interface{}) error {
	var jsonBody bytes.Buffer
	if err := json.NewEncoder(&jsonBody).Encode(request); err != nil {
		return err
	}

	u, err := url.Parse(address)
	if err != nil {
		return err
	}
	u.Path = path

	req, err := http.NewRequest("POST", u.String(), &jsonBody)
	if err != nil {
		return err
	}
	req.SetBasicAuth(LocalNode.ProcessID, SecretToken)

	// Propagate our request ID so that we can trace a request across nodes.
	req.Header.Add("Request-ID", reqid.FromContext(ctx))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", LocalNode.String())

	// TODO(jackson): Add automatic retries (with exponential backoff?)
	resp, err := ctxhttp.Do(ctx, nil, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrStatusCode{
			URL:        u.String(),
			StatusCode: resp.StatusCode,
		}
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return err
		}
	}
	return nil
}
