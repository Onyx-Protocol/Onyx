package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var (
	sshConfig = &ssh.ClientConfig{
		User: "ubuntu",
		Auth: sshAuthMethods(
			os.Getenv("SSH_AUTH_SOCK"),
			os.Getenv("SSH_PRIVATE_KEY"),
		),
	}
)

const usage = `
Command deploy builds and deploys a Chain cmd
to a remote, ubuntu vm. It requires the cmd
name and the ip addr of the target vm. It assumes
the name of the systemd service matches the name
of the cmd. It will attempt to stop the service
before deployment, and restart once the code is
deployed. It uses ssh and also requires either
your private key to be stored in the environment
as SSH_PRIVATE_KEY or ssh-agent to be running.

usage: deploy cmd addr
`

func sshAuthMethods(agentSock, privKeyPEM string) (m []ssh.AuthMethod) {
	conn, sockErr := net.Dial("unix", agentSock)
	key, keyErr := ssh.ParsePrivateKey([]byte(privKeyPEM))
	if sockErr != nil && keyErr != nil {
		log.Println(sockErr)
		log.Println(keyErr)
		log.Fatal("no auth methods found (tried agent and environ)")
	}
	if sockErr == nil {
		m = append(m, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
	}
	if keyErr == nil {
		m = append(m, ssh.PublicKeys(key))
	}
	return m
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println(usage)
		os.Exit(0)
	}
	cmd := os.Args[1]
	addr := os.Args[2]
	bin := mustBuild(cmd)
	might(runOn(addr, "sudo systemctl stop "+cmd))
	must(scpPut(addr, bin, cmd, 0755))
	must(runOn(addr, fmt.Sprintf("sudo mv %s /usr/bin/%s", cmd, cmd)))
	must(runOn(addr, "sudo systemctl start "+cmd))
	log.Println("SUCCESS")
}

func mustBuild(filename string) []byte {
	log.Println("building", filename)

	env := []string{
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Stderr = os.Stderr
	commit, err := cmd.Output()
	must(err)
	commit = bytes.TrimSpace(commit)
	date := time.Now().UTC().Format(time.RFC3339)
	cmd = exec.Command("go", "build",
		"-tags", "http_ok lookback_auth no_reset",
		"-ldflags", "-X main.buildTag=dev -X main.buildDate="+date+" -X main.buildCommit="+string(commit),
		"-o", "/dev/stdout",
		"chain/cmd/"+filename,
	)
	cmd.Env = mergeEnvLists(env, os.Environ())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	must(err)
	log.Printf("%s executable: %d bytes", filename, len(out))
	return out
}

func scpPut(host string, data []byte, dest string, mode int) error {
	log.Printf("scp %d bytes to %s", len(data), dest)
	var client *ssh.Client
	retry(func() (err error) {
		client, err = ssh.Dial("tcp", host+":22", sshConfig)
		return
	})
	defer client.Close()
	s, err := client.NewSession()
	if err != nil {
		return err
	}
	s.Stderr = os.Stderr
	s.Stdout = os.Stderr
	w, err := s.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer w.Close()
		fmt.Fprintf(w, "C%04o %d %s\n", mode, len(data), dest)
		w.Write(data)
		w.Write([]byte{0})
	}()

	return s.Run("/usr/bin/scp -tr .")
}

func runOn(host, sh string, keyval ...string) error {
	if len(keyval)%2 != 0 {
		log.Fatal("odd params", keyval)
	}
	log.Println("run on", host)
	client, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	s, err := client.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	s.Stdout = os.Stderr
	s.Stderr = os.Stderr
	for i := 0; i < len(keyval); i += 2 {
		sh = strings.Replace(sh, "{{"+keyval[i]+"}}", keyval[i+1], -1)
	}
	return s.Run(sh)
}

var errRetry = errors.New("retry")

// retry f until it returns nil.
// wait 500ms in between attempts.
// log err unless it is errRetry.
// after 5 failures, it will call log.Fatal.
// returning errRetry doesn't count as a failure.
func retry(f func() error) {
	for n := 0; n < 5; {
		err := f()
		if err != nil && err != errRetry {
			log.Println("retrying:", err)
			n++
		}
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return
	}
	log.Fatal("too many retries")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func might(err error) {
	if err != nil {
		log.Println(err)
	}
}

// mergeEnvLists merges the two environment lists such that
// variables with the same name in "in" replace those in "out".
// This always returns a newly allocated slice.
func mergeEnvLists(in, out []string) []string {
	out = append([]string(nil), out...)
NextVar:
	for _, inkv := range in {
		k := strings.SplitAfterN(inkv, "=", 2)[0]
		for i, outkv := range out {
			if strings.HasPrefix(outkv, k) {
				out[i] = inkv
				continue NextVar
			}
		}
		out = append(out, inkv)
	}
	return out
}
