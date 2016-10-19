package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chain/net/http/reqid"
)

// Chain-specific header fields
const (
	HeaderBlockchainID = "Blockchain-ID"
	HeaderCoreID       = "Chain-Core-ID"
	HeaderTimeout      = "RPC-Timeout"
)

// ErrWrongNetwork is returned when a peer's blockchain ID differs from
// the RPC client's blockchain ID.
var ErrWrongNetwork = errors.New("connected to a peer on a different network")

// A Client is a Chain RPC client. It performs RPCs over HTTP using JSON
// request and responses. A Client must be configured with a secret token
// to authenticate with other Cores on the network.
type Client struct {
	BaseURL      string
	AccessToken  string
	Username     string
	BuildTag     string
	BlockchainID string
	CoreID       string
}

func (c Client) userAgent() string {
	return fmt.Sprintf("Chain; process=%s; buildtag=%s; blockchainID=%s",
		c.Username, c.BuildTag, c.BlockchainID)
}

// errStatusCode is an error returned when an rpc fails with a non-200
// response code.
type errStatusCode struct {
	URL        string
	StatusCode int
}

func (e errStatusCode) Error() string {
	return fmt.Sprintf("Request to `%s` responded with %d %s",
		e.URL, e.StatusCode, http.StatusText(e.StatusCode))
}

// Call calls a remote procedure on another node, specified by the path.
func (c *Client) Call(ctx context.Context, path string, request, response interface{}) error {
	r, err := c.CallRaw(ctx, path, request)
	if err != nil {
		return err
	}
	defer r.Close()
	err = json.NewDecoder(r).Decode(response)
	return err
}

// CallRaw calls a remote procedure on another node, specified by the path. It
// returns a io.ReadCloser of the raw response body.
func (c *Client) CallRaw(ctx context.Context, path string, request interface{}) (io.ReadCloser, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path

	var bodyReader io.Reader
	if request != nil {
		var jsonBody bytes.Buffer
		if err := json.NewEncoder(&jsonBody).Encode(request); err != nil {
			return nil, err
		}
		bodyReader = &jsonBody
	}

	req, err := http.NewRequest("POST", u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	if c.AccessToken != "" {
		var username, password string
		toks := strings.SplitN(c.AccessToken, ":", 2)
		if len(toks) > 0 {
			username = toks[0]
		}
		if len(toks) > 1 {
			password = toks[1]
		}
		req.SetBasicAuth(username, password)
	}

	// Propagate our request ID so that we can trace a request across nodes.
	req.Header.Add("Request-ID", reqid.FromContext(ctx))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent())
	req.Header.Set(HeaderBlockchainID, c.BlockchainID)
	req.Header.Set(HeaderCoreID, c.CoreID)

	// Propagate our deadline if we have one.
	deadline, ok := ctx.Deadline()
	if ok {
		req.Header.Set(HeaderTimeout, deadline.Sub(time.Now()).String())
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if id := resp.Header.Get(HeaderBlockchainID); c.BlockchainID != "" && id != "" && c.BlockchainID != id {
		resp.Body.Close()
		return nil, ErrWrongNetwork
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, errStatusCode{
			URL:        cleanedURLString(u),
			StatusCode: resp.StatusCode,
		}
	}
	return resp.Body, nil
}

func cleanedURLString(u *url.URL) string {
	var dup url.URL = *u
	dup.User = nil
	return dup.String()
}
