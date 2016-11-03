// Command testbot provides a web server for running Chain integeration tests.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kr/s3"

	"chain/env"
)

var (
	listen      = env.String("LISTEN", ":8080")
	githubToken = os.Getenv("GITHUB_TOKEN")
	slackURL    = os.Getenv("SLACK_WEBHOOK_URL")
	sourcedir   = os.Getenv("CHAIN")
	keys        = s3.Keys{
		AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}
	mu sync.Mutex
)

type pullRequest struct {
	Action string
	PR     struct {
		Number int
		Head   struct {
			Ref string
			Sha string
		}
		StatusesURL string `json:"statuses_url"`
	} `json:"pull_request"`
}

func main() {
	log.SetFlags(log.Lshortfile)
	http.HandleFunc("/pr", prHandler)
	http.HandleFunc("/commit", commitHandler)
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})
	env.Parse()
	log.Println("listening on", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

// handles updates to pull requests
func prHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	req := pullRequest{}
	err := decoder.Decode(&req)
	if err != nil {
		body, err := json.Marshal(map[string]string{"text": fmt.Sprintln("parsing request:", err)})
		if err != nil {
			panic(err)
		}
		postToSlack(body)
		return
	}

	log.Println("pr sha:", req.PR.Head.Sha)
	log.Println("pr action:", req.Action)
	if req.Action == "opened" || req.Action == "synchronize" {
		go func() {
			mu.Lock()
			defer mu.Unlock()

			postToGithub(req.PR.StatusesURL, map[string]string{
				"state":       "pending",
				"description": "Integration tests executing",
				"context":     "chain/testbot",
			})

			var out bytes.Buffer
			sha := req.PR.Head.Sha
			defer uploadToS3(sha, &out)
			defer catch(&out)
			prRef := fmt.Sprintf("pull/%d/head", req.PR.Number)
			runIn(sourcedir, &out, exec.Command("git", "fetch", "origin", prRef), req)
			runIn(sourcedir, &out, exec.Command("git", "clean", "-xdf"), req)
			runIn(sourcedir, &out, exec.Command("git", "checkout", sha, "--"), req)
			runIn(sourcedir, &out, exec.Command("sh", "docker/testbot/tests.sh"), req)
			postToGithub(req.PR.StatusesURL, map[string]string{
				"state":       "success",
				"description": "Integration tests passed",
				"context":     "chain/testbot",
			})
		}()
	}
}

// handles commits to the "main" branch
func commitHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var req struct {
		After, Ref string
	}
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
			defer catch(new(bytes.Buffer))
			select {
			case <-startBenchcore(req.After):
			case <-time.After(2 * time.Minute):
				postToSlackText("starting benchmark timed out for " + req.After)
			}
		}()
	}
}

// ready unblocks when benchcore is no longer
// reading from the filesystem.
func startBenchcore(commit string) (ready <-chan struct{}) {
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
		postToSlackText(fmt.Sprintf("throughput for %s: %.2f tx/s", commit, x.Txs/x.Elapsed))
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

func runIn(dir string, w io.Writer, c *exec.Cmd, req pullRequest) {
	c.Dir = dir
	c.Env = os.Environ()
	c.Stdout = w
	c.Stderr = w
	if err := c.Run(); err != nil {
		log.Printf("error: command run: %s\n", strings.Join(c.Args, " "))
		postToGithub(req.PR.StatusesURL, map[string]string{
			"state":       "failure",
			"description": "Integration tests failed",
			"target_url":  "https://s3.amazonaws.com/chain-qa/testbot/" + req.PR.Head.Sha,
			"context":     "chain/testbot",
		})
		panic(req)
	}
}

func uploadToS3(filename string, logfile io.Reader) {
	log.Println("uploading results to s3")
	req, err := http.NewRequest("PUT", "https://chain-qa.s3.amazonaws.com/testbot/"+filename, logfile)
	if err != nil {
		log.Println("sending request:", err)
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("X-Amz-Acl", "public-read")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Disposition", "inline")
	s3.Sign(req, keys)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	log.Printf("response from aws: %s", resp.Status)
}

func postToGithub(url string, requestBody map[string]string) {
	log.Println("sending results to github")
	b, err := json.Marshal(requestBody)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		log.Println("sending request:", err)
	}
	req.Header.Add("Authorization", "token "+githubToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("sending request:", err)
	}
	defer resp.Body.Close()
	log.Printf("response from github: %s", resp.Status)
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
	log.Printf("response from slack: %s", resp.Status)
}

func catch(w io.Writer) {
	if err := recover(); err != nil {
		switch err := err.(type) {
		case pullRequest:
			fmt.Fprintln(w, err)
		default:
			panic(err)
		}
	}
}
