// Command metricsd provides a daemon for collecting latencies and other
// metrics from cored and uploading them to librato.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/codahale/hdrhistogram"

	"chain/core/rpc"
	"chain/env"
	"chain/log"
	"chain/metrics"
)

const (
	reportTimeout  = 15 * time.Second
	libratoPostURL = "https://metrics-api.librato.com/v1/metrics"
	userAgent      = "chain-metricsd/0.1"
)

var (
	latencyMetricAttributes = attributes{
		Units:     "ms",
		Transform: "x/1000000",
		Summarize: "max",
	}
	period = int64(metrics.Period.Seconds())
)

var (
	coredAddr        = env.String("CORED_ADDR", "http://:1999")
	coredAccessToken = env.String("CORED_ACCESS_TOKEN", "")
	libratoUser      = env.String("LIBRATO_USER", "")
	libratoToken     = env.String("LIBRATO_TOKEN", "")
	metricPrefix     = env.String("METRIC_PREFIX", "cored")
)

func main() {
	env.Parse()

	ctx := context.Background()
	client := &rpc.Client{
		BaseURL:     *coredAddr,
		ProcessID:   userAgent,
		AccessToken: *coredAccessToken,
	}

	// Ensure that we can access cored.
	err := client.Call(ctx, "/health", nil, nil)
	if err != nil {
		log.Fatalkv(ctx, log.KeyError, err)
	}
	log.Printf(ctx, "Successfully pinged cored at %s.", *coredAddr)

	// Periodically, report metrics.
	latestNumRots := make(map[string]int)
	ticker := time.NewTicker(metrics.Period)
	for {
		err := reportMetrics(ctx, client, latestNumRots)
		if err != nil {
			log.Error(ctx, err)
		}
		<-ticker.C
	}
}

func reportMetrics(ctx context.Context, client *rpc.Client, latestNumRots map[string]int) error {
	ctx, cancel := context.WithTimeout(ctx, reportTimeout)
	defer cancel()

	// Query cored for the latest metrics.
	var varsResp debugVarsResponse
	err := client.Call(ctx, "/debug/vars", nil, &varsResp)
	if err != nil {
		return err
	}

	var req libratoMetricsRequest
	req.Source = varsResp.ProcessID
	req.MeasureTime = time.Now().Unix()

	// Add measurements from the runtime memstats.
	// See https://golang.org/pkg/runtime/#MemStats for full
	// documentation on the meaning of these metrics.
	memoryPrefix := *metricPrefix + ".memory."
	req.Gauges = append(req.Gauges, gauge{
		Name:   memoryPrefix + "total",
		Value:  float64(varsResp.Memstats.Alloc),
		Period: period,
		Attr: attributes{
			Units:     "MB",
			Transform: "x/1000000",
			Summarize: "max",
		},
	}, gauge{
		Name:   memoryPrefix + "heap.total",
		Value:  float64(varsResp.Memstats.HeapAlloc),
		Period: period,
		Attr: attributes{
			Units:     "MB",
			Transform: "x/1000000",
			Summarize: "max",
		},
	})
	req.Counters = append(req.Counters, counter{
		Name:  memoryPrefix + "mallocs",
		Value: int64(varsResp.Memstats.Mallocs),
	}, counter{
		Name:  memoryPrefix + "frees",
		Value: int64(varsResp.Memstats.Frees),
	}, counter{
		Name:  memoryPrefix + "gc.total_pause",
		Value: int64(varsResp.Memstats.PauseTotalNs),
		Attr: attributes{
			Units:     "ms",
			Transform: "x/1000000",
			Summarize: "max",
		},
	})

	// Convert the most recent latency histograms into librato gauges.
	for key, latency := range varsResp.Latency {
		// figure out how many buckets have happened since we last
		// recorded data.
		latestRot := latestNumRots[key]
		if latestRot == 0 {
			latestRot = 1
		}
		bucketCount := latency.NumRot - latestRot
		if bucketCount >= len(latency.Buckets) {
			bucketCount = len(latency.Buckets) - 1
		}
		latestNumRots[key] = latency.NumRot

		for b := 1; b <= bucketCount; b++ {
			bucket := latency.Buckets[len(latency.Buckets)-1-b]
			h := hdrhistogram.Import(bucket.Histogram)

			cleanedKey := strings.Replace(strings.Trim(key, "/"), "/", "_", -1)
			name := *metricPrefix + "." + cleanedKey

			req.Gauges = append(req.Gauges, gauge{
				Name:   name + ".qps",
				Value:  float64(h.TotalCount()+bucket.Over) / float64(period),
				Period: period,
				Attr: attributes{
					Units:     "qps",
					Summarize: "sum",
				},
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.mean",
				Value:       h.Mean(),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.p95",
				Value:       float64(h.ValueAtQuantile(95.0)),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.p99",
				Value:       float64(h.ValueAtQuantile(99.0)),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.p999",
				Value:       float64(h.ValueAtQuantile(99.9)),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.max",
				Value:       float64(h.Max()),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			})
		}
	}
	return sendLibratoMetrics(ctx, &req)
}

func sendLibratoMetrics(ctx context.Context, body *libratoMetricsRequest) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", libratoPostURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(*libratoUser, *libratoToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// TODO(jackson): Retry automatically?
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errmsg, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("librato responded with %d:\n%s", resp.StatusCode, errmsg)
	}
	return nil
}

type libratoMetricsRequest struct {
	Source      string    `json:"source,omitempty"`
	MeasureTime int64     `json:"measure_time,omitempty"`
	Counters    []counter `json:"counters,omitempty"`
	Gauges      []gauge   `json:"gauges,omitempty"`
}

type gauge struct {
	Name        string     `json:"name"`
	Value       float64    `json:"value"`
	Period      int64      `json:"period"`
	MeasureTime int64      `json:"measure_time,omitempty"`
	Attr        attributes `json:"attributes"`
}

type counter struct {
	Name        string     `json:"name"`
	Value       int64      `json:"value"`
	MeasureTime int64      `json:"measure_time,omitempty"`
	Attr        attributes `json:"attributes"`
}

type attributes struct {
	Units     string `json:"display_units_long,omitempty"`
	Transform string `json:"display_transform,omitempty"`
	Min       int    `json:"display_min"`
	Summarize string `json:"summarize_function,omitempty"`
}

type debugVarsResponse struct {
	BuildCommit string               `json:"build_commit"`
	BuildDate   string               `json:"build_date"`
	BuildTag    string               `json:"build_tag"`
	Latency     map[string]latencies `json:"latency"`
	Memstats    runtime.MemStats     `json:"memstats"`
	ProcessID   string               `json:"processID"`
}

type latencies struct {
	NumRot  int             `json:"NumRot"`
	Buckets []latencyBucket `json:"Buckets"`
}

type latencyBucket struct {
	Over      int64                  `json:"Over"`
	Timestamp int64                  `json:"Timestamp"`
	Histogram *hdrhistogram.Snapshot `json:"Histogram"`
}
