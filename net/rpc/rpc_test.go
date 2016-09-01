package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

const secretToken = "shhhh, a secret"

func TestRPCCallJSON(t *testing.T) {
	requestBody := map[string]string{
		"hello": "world",
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Inspect the request and ensure that it's what we expect.
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("got=%s; want=application/json", req.Header.Get("Content-Type"))
		}
		if !strings.HasPrefix(req.Header.Get("User-Agent"), "Chain; ") {
			t.Errorf("got=%s; want prefix='Chain; '", req.Header.Get("User-Agent"))
		}
		if req.URL.Path != "/example/rpc/path" {
			t.Errorf("got=%s want=/example/rpc/path", req.URL.Path)
		}
		_, pw, ok := req.BasicAuth()
		if !ok {
			t.Error("no user/password set")
		} else if pw != secretToken {
			t.Errorf("got=%s; want=%s", pw, secretToken)
		}

		decodedRequestBody := map[string]string{}
		if err := json.NewDecoder(req.Body).Decode(&decodedRequestBody); err != nil {
			t.Fatal(err)
		}
		defer req.Body.Close()
		if !reflect.DeepEqual(decodedRequestBody, requestBody) {
			t.Errorf("got=%#v; want=%#v", decodedRequestBody, requestBody)
		}

		// Provide a dummy rpc response
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{
			"response": "example"
		}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	serverURL.User = url.UserPassword("", secretToken)

	response := map[string]string{}
	client := &Client{BaseURL: serverURL.String()}
	err = client.Call(context.Background(), "/example/rpc/path", requestBody, &response)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the response is as we expect.
	if !reflect.DeepEqual(response, map[string]string{"response": "example"}) {
		t.Errorf(`expected map[string]string{"response": "example"}, got %#v`, response)
	}
}

func TestRPCCallError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		http.Error(rw, "a terrible error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL}
	wantErr := errStatusCode{URL: server.URL + "/error", StatusCode: 500}
	err := client.Call(context.Background(), "/error", nil, nil)
	if !reflect.DeepEqual(wantErr, err) {
		t.Errorf("got=%#v; want=%#v", err, wantErr)
	}
}
