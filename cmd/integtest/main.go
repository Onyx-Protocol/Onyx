package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

const (
	pgport = "12345"
)

// This command assumes it has free rein over $HOME/integration.
var (
	home     = os.Getenv("HOME")
	dir      = home + "/integration"
	lockFile = dir + "/lock"
	wkdir    = dir + "/work"
	gobin    = dir + "/bin"
	pgdir    = dir + "/pg"
	pgrun    = dir + "/pgrun" // for socket file
)

var (
	flagT = flag.Duration("t", 15*time.Minute, "abort the test after the given duration")
)

func main() {
	ctx := context.Background()
	log.SetPrefix("integration: ")
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	flag.Usage = usage
	flag.Parse()
	lock() // ensure only one at a time

	ctx, cancel := context.WithTimeout(ctx, *flagT)
	defer cancel()

	if s := pgbin(); s != "" {
		must(os.Setenv("PATH", os.Getenv("PATH")+":"+s))
	}

	must(os.RemoveAll(wkdir))
	must(os.RemoveAll(gobin))
	must(os.RemoveAll(pgdir))

	if flag.NArg() < 1 {
		usage()
	}
	pkg := flag.Arg(0)
	args := flag.Args()[1:]

	// accumulate environment for the test process
	var env []string

	if os.Getenv("CHAIN") == "" {
		env = append(env, "CHAIN="+home+"/go/src/chain")
	}

	setupDB(ctx)
	pgURL := "postgresql:///postgres?host=" + pgrun + "&port=" + pgport
	env = append(env, "DB_URL_TEST="+pgURL) // for chain/database/pg/pgtest

	buildTest(ctx, pkg)

	must(os.MkdirAll(wkdir, 0700))

	_, base := path.Split(pkg)
	fmt.Println(base, strings.Join(args, " "))
	cmd := command(ctx, gobin+"/"+base, args...)
	cmd.Dir = wkdir
	cmd.Env = mergeEnvLists(env, os.Environ())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Printf("%s: %v", base, err)
		panic("cmd failed")
	}
	defer syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	err = cmd.Wait()
	if err != nil {
		log.Printf("%s: %v", base, err)
		panic("cmd failed")
	}
}

func setupDB(ctx context.Context) {
	fmt.Println("initdb", "-D", pgdir)
	err := command(ctx, "initdb", "-D", pgdir).Run()
	if err, ok := err.(*exec.ExitError); ok {
		os.Stderr.Write(err.Stderr)
	}
	if err != nil {
		log.Printf("initdb: %v", err)
		panic("cmd failed")
	}

	var buf bytes.Buffer
	must(configTemplate.Execute(&buf, map[string]string{
		"port":    pgport,
		"sockdir": pgrun,
	}))
	must(ioutil.WriteFile(pgdir+"/postgresql.conf", buf.Bytes(), 0600))

	buf.Reset()
	must(hbaTemplate.Execute(&buf, nil))
	must(ioutil.WriteFile(pgdir+"/pg_hba.conf", buf.Bytes(), 0600))

	fmt.Println("postgres", "-D", pgdir)
	cmd := command(ctx, "postgres", "-D", pgdir)
	cmd.Env = mergeEnvLists([]string{"GOBIN=" + gobin}, os.Environ())
	err = cmd.Start()
	if err, ok := err.(*exec.ExitError); ok {
		os.Stderr.Write(err.Stderr)
	}
	if err != nil {
		log.Printf("go: %v", err)
		panic("cmd failed")
	}

	must(os.MkdirAll(pgrun, 0700))
}

func buildTest(ctx context.Context, pkg string) {
	fmt.Println("go", "install", pkg)
	cmd := command(ctx, "go", "install", pkg)
	cmd.Env = mergeEnvLists([]string{"GOBIN=" + gobin}, os.Environ())
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err, ok := err.(*exec.ExitError); ok {
		os.Stderr.Write(err.Stderr)
	}
	if err != nil {
		log.Printf("go: %v", err)
		panic("cmd failed")
	}
}

func lock() {
	must(os.MkdirAll(dir, 0700))
	f, err := os.Create(lockFile)
	must(err)
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		log.Fatalln("lock:", err)
	}
	// note: do not close f here, retain the lock
	// for the process lifetime
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: integtest [-t duration] [package] [args...]")
	fmt.Fprintln(os.Stderr, "flags:")
	flag.PrintDefaults()
	os.Exit(2)
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

func command(ctx context.Context, name string, arg ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, name, arg...)
	c.SysProcAttr = newSysProcAttr()
	return c
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
