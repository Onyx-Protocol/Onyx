package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"chain/env"
)

const slashLandUsage = `
The /land command takes a git branch reference.
Here is an example using the /land command:

/land [-prv] [-f] feature-x
`

var (
	port          = env.String("PORT", "8080")
	githubToken   = env.String("GITHUB_TOKEN", "")
	org           = env.String("GITHUB_ORG", "chain")
	repo          = env.String("GITHUB_REPO", "chain")
	privRepo      = env.String("GITHUB_REPO_PRIVATE", "chainprv")
	forkRepo      = env.String("GITHUB_REPO_FORK", "chainfork")
	slackChannels = env.StringSlice("SLACK_CHANNEL")
	slackToken    = env.String("SLACK_LAND_TOKEN", "")
	postURL       = env.String("SLACK_POST_URL", "")
)

var landReqs = make(chan *landReq, 10)

type landReq struct {
	userID   string
	userName string
	ref      string
	repo     string
}

func main() {
	log.SetFlags(log.Lshortfile)
	env.Parse()

	err := configGit()
	if err != nil {
		log.Fatalln(err)
	}
	err = writeNetrc()
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
	repo := *repo
	if len(a) >= 1 && a[0] == "-prv" {
		repo = *privRepo
		a = a[1:]
	} else if len(a) >= 1 && a[0] == "-f" {
		repo = *forkRepo
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
		repo:     repo,
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

	gopath := "/tmp/land"
	landdir := gopath + "/src/" + req.repo

	fetch(landdir, req.ref, req.repo)
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
		Base      struct{ Ref string }
	}
	err = doGithubReq("GET", "repos/"+*org+"/"+req.repo+"/pulls/"+pr, nil, &prState)
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
	if prState.Mergeable != nil && !*prState.Mergeable {
		sayf("<@%s|%s> failed to land %s: branch has conflicts",
			req.userID,
			req.userName,
			req.ref,
		)
		return
	}

	// base branch e.g. origin/main, origin/chain-core-server-1.1.x
	cmd := dirCmd(landdir, "git", "rebase", "origin/"+prState.Base.Ref)
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

	err = commitRevIDs(landdir, prState.Base.Ref)
	if err != nil {
		sayf("<@%s|%s> failed to land %s: could not commit revision id: %s",
			req.userID,
			req.userName,
			req.ref,
			err,
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

	success := waitForSuccessfulStatus(req, req.repo, commit)
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
		MergeMethod   string `json:"merge_method"`
	}{prState.Title, wrapMessage(body, 75), commit, "squash"}
	var mergeResp struct {
		Merged  bool
		Message string
	}
	err = doGithubReq("PUT", fmt.Sprintf("repos/%s/%s/pulls/%s/merge", *org, req.repo, pr), mergeReq, &mergeResp)
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
	fetch(landdir, "main", req.repo)
	runIn(landdir, exec.Command("git", "branch", "-D", req.ref))
}

func commitRevIDs(landdir, baseBranch string) error {
	revID, err := revID(landdir, baseBranch)
	if err != nil {
		return err
	}

	for name, tpl := range revIDLang {
		var body bytes.Buffer
		err = tpl.Execute(&body, revID)
		if err != nil {
			return err
		}
		path := filepath.Join(landdir, name)
		err = ioutil.WriteFile(path, body.Bytes(), 0666)
		if err != nil {
			return err
		}
	}

	// We have to add the files here for a weird reason:
	// Running 'git commit' (without --allow-empty) will fail when
	// the file contents haven't changed, regardless of the files'
	// time stamps. We want to detect this situation ahead of time
	// and skip committing when the tree is clean.
	// Unfortunately, diff-index *does* consider timestamps when
	// computing the diff; fortunately, running 'git add' fixes this
	// and causes the behavior of diff-index to match what commit
	// looks for in its "nothing to commit" error message.
	cmd := dirCmd(landdir, "git", "add", "generated")
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	if isClean(landdir) {
		// Avoid adding empty commits here if the revid hasn't
		// changed since the last rebase. This way we don't
		// have to wait on CI to run, we can land immediately.
		return nil
	}

	cmd = dirCmd(landdir, "git", "commit", "-m", "auto rev id")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeNetrc() error {
	p := filepath.Clean(os.Getenv("HOME") + "/.netrc")
	s := "machine github.com login chainbot password " + *githubToken + "\n"
	return ioutil.WriteFile(p, []byte(s), 0600)
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

func waitForSuccessfulStatus(req *landReq, repo, commitSHA string) bool {
	start := time.Now()
	for {
		if time.Since(start) > 4*time.Minute {
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
		err := doGithubReq("GET", fmt.Sprintf("repos/%s/%s/commits/%s/status", *org, repo, req.ref), nil, &statusResp)
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

func isClean(dir string) bool {
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err == nil
}
