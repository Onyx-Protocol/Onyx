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
	"strings"
	"time"

	"chain/env"
)

const slashLandUsage = `
The /land command takes a git branch reference.
Here is an example using the /land command:

/land [-prv] feature-x
`

var (
	port          = env.String("PORT", "8080")
	githubToken   = env.String("GITHUB_TOKEN", "")
	org           = env.String("GITHUB_ORG", "chain")
	repo          = env.String("GITHUB_REPO", "chain")
	privRepo      = env.String("GITHUB_REPO_PRIVATE", "chainprv")
	slackChannels = env.StringSlice("SLACK_CHANNEL")
	slackToken    = env.String("SLACK_LAND_TOKEN", "")
	postURL       = env.String("SLACK_POST_URL", "")
)

var landReqs = make(chan *landReq, 10)

type landReq struct {
	userID   string
	userName string
	ref      string
	private  bool
}

func main() {
	log.SetFlags(log.Lshortfile)
	env.Parse()

	err := configGit()
	if err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/slash", slashLand)
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
	})

	go lander()

	say("Ready for action!")
	err = http.ListenAndServe(":"+*port, nil)
	log.Fatalln(err)
}

func slashLand(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("token") != *slackToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !contains(r.FormValue("channel_id"), *slackChannels) {
		http.Error(w, fmt.Sprintf("%s not enabled in this channel.", r.FormValue("command")), 400)
		return
	}
	a := strings.Fields(r.FormValue("text"))
	private := false
	if len(a) >= 1 && a[0] == "-prv" {
		private = true
		a = a[1:]
	}
	if len(a) != 1 {
		http.Error(w, slashLandUsage, 422)
		return
	}
	landReqs <- &landReq{
		ref:      a[0],
		userID:   r.FormValue("user_id"),
		userName: r.FormValue("user_name"),
		private:  private,
	}
	sayf("<@%s|%s> is attempting to land %s",
		r.FormValue("user_id"),
		r.FormValue("user_name"),
		a[0],
	)
}

func lander() {
	for req := range landReqs {
		land(req)
	}
}

func land(req *landReq) {
	defer catch()

	repo := *repo
	if req.private {
		repo = *privRepo
	}

	gopath := "/tmp/land"
	landdir := gopath + "/src/" + repo

	fetch(landdir, req.ref, repo)
	commit := string(bytes.TrimSpace(runOutput(landdir, exec.Command("git", "rev-parse", "HEAD"))))

	prBits, err := pipeline(
		dirCmd(landdir, "git", "ls-remote", "origin", `refs/pull/*/head`),
		exec.Command("fgrep", commit),
		exec.Command("cut", "-d/", "-f3"),
	)
	if err != nil {
		sayf("<@%s|%s> failed to land %s: could not find open pull request",
			req.userID,
			req.userName,
			req.ref,
		)
		return
	}
	pr := string(bytes.TrimSpace(prBits))

	var prState struct {
		Title     string
		Body      string
		Merged    bool
		Mergeable *bool
	}
	err = doGithubReq("GET", "repos/"+*org+"/"+repo+"/pulls/"+pr, nil, &prState)
	if err != nil {
		sayf("<@%s|%s> failed to land %s: error fetching github status",
			req.userID,
			req.userName,
			req.ref,
		)
	}
	if prState.Merged {
		sayf("<@%s|%s> %s has already landed",
			req.userID,
			req.userName,
			req.ref,
		)
		return
	}
	if prState.Mergeable != nil && *prState.Mergeable == false {
		sayf("<@%s|%s> failed to land %s: branch has conflicts",
			req.userID,
			req.userName,
			req.ref,
		)
		return
	}

	cmd := dirCmd(landdir, "git", "rebase", "origin/main")
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		cmd = dirCmd(landdir, "git", "rebase", "--abort")
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		sayf("<@%s|%s> failed to land %s: branch has conflicts",
			req.userID,
			req.userName,
			req.ref,
		)
		return
	}

	cmd = dirCmd(landdir, "git", "filter-branch", "-f", "--env-filter", `
		export GIT_COMMITTER_NAME=$GIT_AUTHOR_NAME
		export GIT_COMMITTER_EMAIL=$GIT_AUTHOR_EMAIL
	`, "--", "origin/main..", req.ref)
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		sayf("<@%s|%s> failed to land %s: could not fix commiter name/email",
			req.userID,
			req.userName,
			req.ref,
		)
		return
	}

	cmd = dirCmd(landdir, "git", "push", "origin", req.ref, "-f")
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		sayf("<@%s|%s> failed to land %s: could not push rebase (%s)",
			req.userID,
			req.userName,
			req.ref,
			err,
		)
		return
	}

	commit = string(bytes.TrimSpace(runOutput(landdir, exec.Command("git", "rev-parse", "HEAD"))))

	success := waitForSuccessfulStatus(req, commit)
	if !success {
		return
	}

	body := prState.Body
	if body != "" {
		body += "\n\n"
	}
	body += fmt.Sprintf("Closes #%s\n", pr)
	mergeReq := struct {
		CommitTitle   string `json:"commit_title"`
		CommitMessage string `json:"commit_message"`
		SHA           string `json:"sha"`
		Squash        bool   `json:"squash"`
	}{prState.Title, wrapMessage(body, 75), commit, true}
	var mergeResp struct {
		Merged  bool
		Message string
	}
	err = doGithubReq("PUT", fmt.Sprintf("repos/%s/%s/pulls/%s/merge", *org, repo, pr), mergeReq, &mergeResp)
	if err != nil {
		sayf("<@%s|%s> failed to land %s: could not merge pull request (%s)",
			req.userID,
			req.userName,
			req.ref,
			err,
		)
		return
	}
	if !mergeResp.Merged {
		sayf("<@%s|%s> failed to land %s: could not merge pull request (%s)",
			req.userID,
			req.userName,
			req.ref,
			mergeResp.Message,
		)
		return
	}

	runIn(landdir, exec.Command("git", "push", "origin", ":"+req.ref))
	fetch(landdir, "main", repo)
	runIn(landdir, exec.Command("git", "branch", "-D", req.ref))
}

func configGit() error {
	cmd := exec.Command("git", "config", "--global", "user.email", "ops@chain.com")
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	cmd = exec.Command("git", "config", "--global", "user.name", "chainbot")
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return err
}

func wrapMessage(msg string, limit int) string {
	b := make([]byte, 0, len(msg))
	limit++ // accounts for whitespace from SplitAfter

	for _, line := range strings.SplitAfter(msg, "\n") {
		lineLen := 0
		for _, word := range strings.SplitAfter(line, " ") {
			if lineLen+len(word) > limit && lineLen > 0 {
				lineLen = 0
				b = bytes.TrimRight(b, " ")
				b = append(b, '\n')
			}
			b = append(b, word...)
			lineLen += len(word)
		}
	}
	return string(b)
}

func waitForSuccessfulStatus(req *landReq, commitSHA string) bool {
	start := time.Now()
	for {
		if time.Since(start) > 3*time.Minute {
			sayf("<@%s|%s> failed to land %s: timed out waiting for build status",
				req.userID,
				req.userName,
				req.ref,
			)
			return false
		}

		var statusResp struct {
			State, SHA string
		}
		err := doGithubReq("GET", fmt.Sprintf("repos/%s/%s/commits/%s/status", *org, *repo, req.ref), nil, &statusResp)
		if err != nil || statusResp.State == "" {
			sayf("<@%s|%s> failed to land %s: error fetching github status",
				req.userID,
				req.userName,
				req.ref,
			)
			return false
		}
		if statusResp.State == "failure" {
			sayf("<@%s|%s> failed to land %s: build failed",
				req.userID,
				req.userName,
				req.ref,
			)
			return false
		}
		if statusResp.State == "success" && statusResp.SHA == commitSHA {
			break
		}
		time.Sleep(15 * time.Second)
	}
	return true
}

func doGithubReq(method, path string, body, x interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, "https://api.github.com/"+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "token "+*githubToken)
	req.Header.Add("Accept", "application/vnd.github.polaris-preview+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(x)
}

func fetch(dir, ref, repo string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		clone(dir, ref, repo)
		return
	}
	runIn(dir, exec.Command("git", "fetch", "origin"))
	runIn(dir, exec.Command("git", "clean", "-xdf"))
	runIn(dir, exec.Command("git", "checkout", ref, "--"))
	runIn(dir, exec.Command("git", "reset", "--hard", "origin/"+ref))
}

func clone(dir, ref, repo string) {
	c := exec.Command("git", "clone",
		"--branch="+ref,                                          // Check out branch 'ref'
		fmt.Sprintf("https://github.com/%s/%s.git/", *org, repo), // from remote 'url'
		dir, // into directory 'dir'.
	)
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		panic(fmt.Errorf("%s: %v", strings.Join(c.Args, " "), err))
	}
}
