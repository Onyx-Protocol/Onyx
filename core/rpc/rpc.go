// Package rpc implements Chain Core's RPC client.
package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chain/errors"
	"chain/net/http/httperror"
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
	ProcessID    string
	Version      string
	BlockchainID string
	CoreID       string

	// If set, Client is used for outgoing requests.
	// TODO(kr): make this required (crash on nil)
	Client *http.Client
}

func (c Client) userAgent() string {
	return fmt.Sprintf("Chain; process=%s; version=%s; blockchainID=%s",
		c.ProcessID, c.Version, c.BlockchainID)
}

// ErrStatusCode is an error returned when an rpc fails with a non-200
// response code.
type ErrStatusCode struct {
	URL        string
	StatusCode int
	ErrorData  *httperror.Response
}

func (e ErrStatusCode) Error() string {
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
	if response != nil {
		err = errors.Wrap(json.NewDecoder(r).Decode(response))
	}
	return err
}

// CallRaw calls a remote procedure on another node, specified by the path. It
// returns a io.ReadCloser of the raw response body.
func (c *Client) CallRaw(ctx context.Context, path string, request interface{}) (io.ReadCloser, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	u.Path = path

	var bodyReader io.Reader
	if request != nil {
		var jsonBody bytes.Buffer
		if err := json.NewEncoder(&jsonBody).Encode(request); err != nil {
			return nil, errors.Wrap(err)
		}
		bodyReader = &jsonBody
	}

	req, err := http.NewRequest("POST", u.String(), bodyReader)
	if err != nil {
		return nil, errors.Wrap(err)
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

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil && ctx.Err() != nil { // check if it timed out
		return nil, errors.Wrap(ctx.Err())
	} else if err != nil {
		return nil, errors.Wrap(err)
	}

	if id := resp.Header.Get(HeaderBlockchainID); c.BlockchainID != "" && id != "" && c.BlockchainID != id {
		resp.Body.Close()
		return nil, errors.Wrap(ErrWrongNetwork)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()

		resErr := ErrStatusCode{
			URL:        cleanedURLString(u),
			StatusCode: resp.StatusCode,
		}

		// Attach formatted error message, if available
		if errData, ok := httperror.Parse(resp.Body); ok {
			resErr.ErrorData = errData
		}

		return nil, resErr
	}

	return resp.Body, nil
}

func cleanedURLString(u *url.URL) string {
	var dup url.URL = *u
	dup.User = nil
	return dup.String()
}
