package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"chain/env"
)

var (
	listen    = env.String("LISTEN", ":8080")
	slackURL  = os.Getenv("SLACK_WEBHOOK_URL")
	sourcedir = os.Getenv("CHAIN")
	mu        sync.Mutex
)

type Req struct {
	Ref    string
	After  string
	Log    string
	Commit struct {
		Message string
		URL     string
		Author  struct {
			Username string
		}
	} `json:"head_commit"`
}

func main() {
	log.SetFlags(log.Lshortfile)
	http.HandleFunc("/push", handler)
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})

	log.Println("listening on", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	req := Req{}
	err := decoder.Decode(&req)
	if err != nil {
		body, err := json.Marshal(map[string]string{"text": fmt.Sprintln("parsing request:", err)})
		if err != nil {
			panic(err)
		}
		postToSlack(body)
		return
	}

	log.Println("ref pushed:", req.Ref)
	if req.Ref == "refs/heads/main" {
		go func() {
			mu.Lock()
			defer mu.Unlock()
			defer catch()
			runIn(sourcedir, exec.Command("git", "fetch", "origin"), req)
			runIn(sourcedir, exec.Command("git", "clean", "-xdf"), req)
			runIn(sourcedir, exec.Command("git", "checkout", req.After, "--"), req)
			runIn(sourcedir, exec.Command("git", "reset", "--hard", req.After), req)
			runIn(sourcedir, exec.Command("./bin/run-tests"), req)
			postToSlack(buildBody(req))
			select {
			case <-startBenchcore(req.Ref):
			case <-time.After(2 * time.Minute):
				postToSlackText("starting benchmark timed out for " + req.Ref)
			}
		}()
	}
}

// ready unblocks when benchcore is no longer
// reading from the filesystem.
func startBenchcore(ref string) (ready <-chan struct{}) {
	ch := make(chan struct{})
	go func() {
		var errbuf bytes.Buffer
		c := exec.Command("benchcore", "IouSettlement.java")
		c.Dir = filepath.Join(sourcedir, "perf")
		c.Stderr = io.MultiWriter(&errbuf, &signalWriter{
			target: []byte("READY, done with local filesystem"),
			done:   ch,
		})
		stats, err := c.Output()
		// TODO(kr): save log in s3
		if err != nil {
			postToSlackText("benchcore error: " + err.Error())
			return
		}
		// TODO(kr): save stats in s3 (not just slack)
		var x struct {
			Elapsed float64 `json:"elapsed_ms"`
			Txs     float64
		}
		err = json.Unmarshal(stats, &x)
		if err != nil {
			postToSlackText("benchcore error: " + err.Error())
			return
		}
		x.Elapsed = x.Elapsed / 1000 // ms -> s
		postToSlackText(fmt.Sprintf("throughput for %s: %.2f tx/s", ref, x.Txs/x.Elapsed))
	}()
	return ch
}

type signalWriter struct {
	target []byte
	done   chan<- struct{}

	buf []byte // lazily initialized to 2x len(target)
	w   int    // bytes written to 2nd half of buf
}

func (w *signalWriter) Write(p []byte) (int, error) {
	if w.done == nil {
		return len(p), nil
	}
	if w.buf == nil {
		w.buf = make([]byte, 2*len(w.target))
	}

	t := len(w.target)
	bufa := w.buf[:t]
	bufb := w.buf[t:]

	for r := 0; r < len(p); {
		n := copy(bufb[w.w:], p[r:])
		w.w += n
		r += n
		if w.w == t {
			copy(bufa, bufb)
			w.w = 0
		}
		if bytes.Contains(w.buf[:t+w.w], w.target) {
			close(w.done)
			w.done = nil
			break
		}
	}
	return len(p), nil
}

func buildBody(req Req) []byte {
	var color, result string
	if req.Log == "" {
		color = "good"
		result = "passed :thumbsup:"
	} else {
		color = "danger"
		result = "failed :thumbsdown:"
	}

	// Only display the commit message header
	split := strings.Split(req.Commit.Message, "\n")
	msg := split[0]
	buffer := `{
		"attachments": [
			{
				"color": "` + color + `",
				"text": "Integration tests ` + result + `",
				"fields": [
					{
						"title": "Commit",
						"value": "<` + req.Commit.URL + `|` + msg + `>",
						"short": false
					},
					{
						"title": "Author",
						"value": "<https://github.com/` + req.Commit.Author.Username + `|` + req.Commit.Author.Username + `>",
						"short": false
					}
				]
			}`

	if req.Log == "" {
		// end json buffer
		buffer += `]}`
	} else {
		// add the error log
		buffer += `,{
			"text": "` + html.EscapeString(req.Log) + `",
			"mrkdwn_in": [
				"text"
			]
		}]}`
	}
	return []byte(buffer)
}

func runIn(dir string, c *exec.Cmd, req Req) {
	var outbuf, errbuf bytes.Buffer
	c.Dir = dir
	c.Env = os.Environ()
	c.Stdout = &outbuf
	c.Stderr = &errbuf
	if err := c.Run(); err != nil {
		req.Log = fmt.Sprintf("Command run: `%s`\n%s", strings.Join(c.Args, " "), errbuf.String())
		panic(buildBody(req))
	}
}

func postToSlackText(s string) {
	b, err := json.Marshal(map[string]string{"text": s})
	if err != nil {
		panic(err)
	}
	postToSlack(b)
}

func postToSlack(b []byte) {
	log.Println("sending results to slack")
	resp, err := http.Post(slackURL, "application/json", bytes.NewReader(b))
	if err != nil {
		log.Println("sending request:", err)
	}
	defer resp.Body.Close()
}

func catch() {
	if err := recover(); err != nil {
		switch err := err.(type) {
		case []byte:
			postToSlack(err)
		default:
			panic(err)
		}
	}
}
