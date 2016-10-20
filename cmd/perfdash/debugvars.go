package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/codahale/hdrhistogram"
)

type debugVars struct {
	Raw     *json.RawMessage
	Latency map[string]struct {
		Buckets []struct {
			Over      int
			Max       int64
			Histogram hdrhistogram.Snapshot
		}
	}
}

var (
	debugVarMu   sync.Mutex
	debugVarData = make(map[int]*debugVars)
	debugVarNext int
)

func getDebugVars(i int) *debugVars {
	debugVarMu.Lock()
	defer debugVarMu.Unlock()
	return debugVarData[i]
}

func fetchDebugVars(baseURL, token string) (int, *debugVars, error) {
	v := new(debugVars)

	req, err := http.NewRequest("GET", strings.TrimRight(baseURL, "/")+"/debug/vars", nil)
	if err != nil {
		return 0, nil, err
	}
	if i := strings.Index(token, ":"); i >= 0 {
		req.SetBasicAuth(token[:i], token[i+1:])
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	err = json.Unmarshal(b, v)
	if err != nil {
		return 0, nil, err
	}

	v.Raw = (*json.RawMessage)(&b)

	debugVarMu.Lock()
	n := debugVarNext
	debugVarNext++
	debugVarData[n] = v
	debugVarMu.Unlock()

	return n, v, err
}
