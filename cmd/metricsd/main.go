// Command metricsd provides a daemon for collecting latencies and other
// metrics from cored and uploading them to librato.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/codahale/hdrhistogram"

	"chain/env"
	"chain/log"
	"chain/metrics"
	"chain/net/rpc"
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
		Username:    userAgent,
		AccessToken: *coredAccessToken,
	}

	// Ensure that we can access cored.
	err := client.Call(ctx, "/health", nil, nil)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}
	log.Messagef(ctx, "Successfully pinged cored at %s.", *coredAddr)

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

	// Convert the histograms into librato gauges.
	var req libratoMetricsRequest
	req.Source = varsResp.ProcessID
	for endpoint, latency := range varsResp.Latency {
		// figure out how many buckets have happened since we last
		// recorded data.
		latestRot := latestNumRots[endpoint]
		if latestRot == 0 {
			latestRot = 1
		}
		bucketCount := latency.NumRot - latestRot
		if bucketCount >= len(latency.Buckets) {
			bucketCount = len(latency.Buckets) - 1
		}
		latestNumRots[endpoint] = latency.NumRot

		for b := 1; b <= bucketCount; b++ {
			bucket := latency.Buckets[len(latency.Buckets)-1-b]
			h := hdrhistogram.Import(bucket.Histogram)

			cleanedEndpoint := strings.Replace(strings.Trim(endpoint, "/"), "/", "_", -1)
			name := *metricPrefix + ".rpc." + cleanedEndpoint

			req.Gauges = append(req.Gauges, gauge{
				Name:        name + ".latency.mean",
				Value:       int64(h.Mean()),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.p95",
				Value:       h.ValueAtQuantile(95.0),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.p99",
				Value:       h.ValueAtQuantile(99.0),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.p999",
				Value:       h.ValueAtQuantile(99.9),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			}, gauge{
				Name:        name + ".latency.max",
				Value:       h.Max(),
				Attr:        latencyMetricAttributes,
				Period:      period,
				MeasureTime: bucket.Timestamp,
			})
		}
	}
	return sendLibratoMetrics(ctx, &req)
}

func sendLibratoMetrics(ctx context.Context, body *libratoMetricsRequest) error {
	if len(body.Gauges) == 0 {
		return nil
	}

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
		return fmt.Errorf("librato responded with %d", resp.StatusCode)
	}
	return nil
}

type libratoMetricsRequest struct {
	Source string  `json:"source,omitempty"`
	Gauges []gauge `json:"gauges,omitempty"`
}

type gauge struct {
	Name        string     `json:"name"`
	Value       int64      `json:"value"`
	Period      int64      `json:"period"`
	MeasureTime int64      `json:"measure_time"`
	Attr        attributes `json:"attributes"`
}

type attributes struct {
	Units     string `json:"display_units_long,omitempty"`
	Transform string `json:"display_transform,omitempty"`
	Min       int    `json:"display_min"`
	Summarize string `json:"summarize_function,omitempty"`
}

type debugVarsResponse struct {
	BuildCommit string               `json:"buildcommit"`
	BuildDate   string               `json:"builddate"`
	BuildTag    string               `json:"buildtag"`
	Latency     map[string]latencies `json:"latency"`
	ProcessID   string               `json:"processID"`
}

type latencies struct {
	NumRot  int             `json:"NumRot"`
	Buckets []latencyBucket `json:"Buckets"`
}

type latencyBucket struct {
	Over      uint64                 `json:"Over"`
	Timestamp int64                  `json:"Timestamp"`
	Histogram *hdrhistogram.Snapshot `json:"Histogram"`
}
