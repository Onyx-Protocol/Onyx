package etcdname

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"

	"chain-stealth/log"
)

var (
	etcd    client.Client
	initErr error
)

func init() {
	u := os.Getenv("ETCD_URLS")
	if u == "" {
		return
	}

	cfg := client.Config{
		Endpoints:               strings.Split(u, ","),
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	etcd, initErr = client.New(cfg)
}

// LookupHost looks up the given host using the configured etcd cluster, if any.
// It retrieves the address for a host by checking etcd's services directory, and
// returns an array of the provided host's addresses.
//
// when querying etcd, LookupHost expects the response from the services directory
// to be one of three things:
//		1. an IP address or comma-delimted string of IP addresses
//       (like "127.0.0.1,128.0.0.1")
//		2. an etcd key (like "/fooDB/primary")
// 		3. an etcd key and a JSON pointer, separated with a # (like
//       "/fooDB/primary#primaryIP")
//
// If the services directory contains an IP address or string of addresses,
// LookupHost returns them. Otherwise, it checks the provided etcd key.
// If no JSON pointer is provided, LookupHost expects the value at that etcd key
// to be a string (a single address, or a comma-delimited string of addresses).
// Otherwise, it expects JSON, and it will parse that JSON using the provided JSON pointer.
//
// For more on JSON pointers, see RFC 6901: https://tools.ietf.org/html/rfc6901.
func LookupHost(host string) ([]string, error) {
	ctx := context.TODO()
	if initErr != nil {
		log.Error(ctx, initErr)
		return nil, initErr
	} else if etcd == nil {
		return nil, errors.New("etcd is not configured")
	}

	kapi := client.NewKeysAPI(etcd)
	resp, err := kapi.Get(ctx, "/services/"+host, nil)
	if err != nil {
		return nil, err
	}

	if resp.Node.Value[0] != '/' {
		// This isn't an etcd key.
		return strings.Split(resp.Node.Value, ","), nil
	}

	etcdKey, jsonPointer := splitPointer(resp.Node.Value)
	if err != nil {
		return nil, err
	}

	resp, err = kapi.Get(ctx, etcdKey, nil)
	if err != nil {
		return nil, err
	}

	if jsonPointer == "" {
		// If there's no JSON Pointer, just return the response from etcd.
		return strings.Split(resp.Node.Value, ","), nil
	}

	addrs, err := unmarshalFromPointer([]byte(resp.Node.Value), jsonPointer)
	if err != nil {
		return nil, err
	}

	as := strings.Split(addrs, ",")
	for _, a := range as {
		if net.ParseIP(a) == nil {
			return nil, errors.New("bad address: " + a)
		}
	}

	return as, nil
}

// splitPointer splits a response from etcd's services directory into
// an etcd key and a JSON Pointer. If there isn't a JSON Pointer,
// parsePath will return an empty string as the JSON Pointer.
// If there's more than one JSON Pointer, parsePath will return
// an error.
func splitPointer(path string) (etcdKey, jsonPointer string) {
	if idx := strings.IndexByte(path, '#'); idx >= 0 {
		return path[:idx], path[idx+1:]
	}
	return path, ""
}

// unmarshalFromPointer parses a value inside the JSON-encoded data using the
// given pointer. See the JSON Pointer RFC (https://tools.ietf.org/html/rfc6901) for more.
func unmarshalFromPointer(data []byte, pointer string) (string, error) {
	var v interface{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return "", err
	}

	res := followJSONPointer(v, strings.Split(pointer, "/"))
	if res == "" {
		return "", errors.New("could not find value at that JSON pointer")
	}

	return res, nil
}

// followJSONPointer traverses the result of json.Unmarshal, looking for the value
// specified by the pointer. If it cannot find anything, it returns the empty string.
func followJSONPointer(v interface{}, pointer []string) string {
	if len(pointer) == 0 {
		str, _ := v.(string)
		return str
	}

	switch v := v.(type) {
	case map[string]interface{}:
		// this is an object

		k := pointer[0]
		if strings.Contains(k, "~") {
			k = strings.Replace(k, "~1", "/", -1)
			k = strings.Replace(k, "~0", "~", -1)
		}

		return followJSONPointer(v[k], pointer[1:])

	case []interface{}:
		// this is an array
		i, err := strconv.Atoi(pointer[0])
		if err != nil || i >= len(v) || i < 0 {
			return ""
		}
		return followJSONPointer(v[i], pointer[1:])
	}

	return ""
}
