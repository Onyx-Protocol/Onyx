package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"chain/net/http/reqid"
)

// A Client is a Chain RPC client. It performs RPCs over HTTP using JSON
// request and responses. A Client must be configured with a secret token
// to authenticate with other Cores on the network.
type Client struct {
	BaseURL  string
	Username string
	BuildTag string
}

func (c Client) userAgent() string {
	return fmt.Sprintf("Chain; process=%s; buildtag=%s",
		c.Username, c.BuildTag)
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
func (c *Client) Call(ctx context.Context, path string, request, response interface{}) error {
	var jsonBody bytes.Buffer
	if err := json.NewEncoder(&jsonBody).Encode(request); err != nil {
		return err
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	u.Path = path

	req, err := http.NewRequest("POST", u.String(), &jsonBody)
	if err != nil {
		return err
	}

	var username, password string
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	req.SetBasicAuth(username, password)

	// Propagate our request ID so that we can trace a request across nodes.
	req.Header.Add("Request-ID", reqid.FromContext(ctx))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent())

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
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
