package librato

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/codahale/metrics"
)

var (
	// Prefix will be added to the beginning of each metric name
	// before SampleMetrics sends it to librato.
	//
	// You should set it to the name of your program.
	// For example,
	//   librato.Prefix = "myprogram."
	Prefix string

	// Source is used as the "source" parameter in the Librato API.
	// It is typically a process ID or machine name.
	// For example,
	//   librato.Source = os.Getenv("DYNO")
	Source string

	// URL will be used by SampleMetrics
	// to connect to the Librato API.
	// Field User will be used for HTTP Basic authentication.
	URL *url.URL
)

// Leave http.DefaultClient alone, and set our own timeout here.
var httpClient = &http.Client{Timeout: 1 * time.Second}

// SampleMetrics periodically samples all defined metrics and
// sends them to the given Librato API URL.
func SampleMetrics(period time.Duration) {
	user := URL.User.Username()
	pass, _ := URL.User.Password()
	URL.User = nil
	url := URL.String()

	for now := range time.Tick(period) {
		c, g := metrics.Snapshot()
		var gauges struct {
			G []metric `json:"gauges"`
		}
		// NOTE(kr): this is a little sloppy.
		// We combine Counters and Gauges
		// into a single namespace.
		// One could define a Counter and Gauge
		// with the same name –
		// here only the Gauge would be visible.
		for k, v := range c {
			gauges.G = append(gauges.G, newMetric(k, int64(v), now))
		}
		for k, v := range g {
			gauges.G = append(gauges.G, newMetric(k, v, now))
		}
		err := post(gauges, url, user, pass)
		if err != nil {
			log.Println(err)
		}
	}
}

type metric struct {
	Name   string `json:"name"`
	Val    int64  `json:"value"`
	Time   int64  `json:"measure_time"`
	Source string `json:"source,omitempty"`
	Auth   string `json:"-"`
	Attr   struct {
		Units     string `json:"display_units_long,omitempty"`
		Transform string `json:"display_transform,omitempty"`
		Min       int    `json:"display_min"`
		Summarize string `json:"summarize_function,omitempty"`
	} `json:"attributes"`
}

func newMetric(name string, val int64, now time.Time) (m metric) {
	m.Name = Prefix + name
	m.Source = Source
	m.Time = now.Unix()
	if strings.Contains(name, "duration") {
		m.Attr.Units = "ms"
		m.Attr.Transform = "x/1000000"
		m.Attr.Min = 0
		m.Attr.Summarize = "max"
	}
	m.Val = val
	return m
}

func post(v interface{}, url, user, pass string) error {
	j, err := json.Marshal(v)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(j)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "chain-engineering-librato/0")
	req.Header.Add("Connection", "Keep-Alive")
	req.SetBasicAuth(user, pass)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var m string
		s, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			m = fmt.Sprintf("code=%d", resp.StatusCode)
		} else {
			m = fmt.Sprintf("code=%d resp=body=%s req-body=%s",
				resp.StatusCode, s, body)
		}
		return errors.New(m)
	}
	return nil
}
