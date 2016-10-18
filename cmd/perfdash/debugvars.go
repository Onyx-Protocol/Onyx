package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/codahale/hdrhistogram"
)

type debugVars struct {
	Latency map[string]struct {
		Buckets []struct {
			Over      int
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

func fetchDebugVars(baseURL string) (int, *debugVars, error) {
	v := new(debugVars)
	resp, err := http.Get(strings.TrimRight(baseURL, "/") + "/debug/vars")
	if err != nil {
		return 0, nil, err
	}
	err = json.NewDecoder(resp.Body).Decode(v)

	debugVarMu.Lock()
	n := debugVarNext
	debugVarNext++
	debugVarData[n] = v
	debugVarMu.Unlock()

	return n, v, err
}
