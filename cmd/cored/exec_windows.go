package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"chain/core"
)

// We have to do some tricks here because we can't use exec.
// Instead of exec, we have the first cored spawn a child,
// and when we would normally exec, instead the child will
// exit and the parent will restart it.
func maybeMonitorIfOnWindows() {
	if !inChild() {
		monitor() // never returns
	}
}

func inChild() bool {
	return os.Args[0] == "coredchild"
}

func monitor() {
	self, err := os.Executable()
	if err != nil {
		panic(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	var lastExitCode uint32 // plumbing for reset requests

	for {
		cmd := exec.Command(self, os.Args...)
		cmd.Args[0] = "coredchild"
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		env := "RESET=" + core.WinResetCodeToEnv[lastExitCode]
		cmd.Env = mergeEnvLists([]string{env}, os.Environ())
		err := cmd.Start()
		if err != nil {
			log.Fatalln(err)
		}

		wait := make(chan error, 1)
		go func() { wait <- cmd.Wait() }()

		select {
		case v := <-sig:
			cmd.Process.Signal(v)
			err = <-wait
			if err != nil {
				log.Fatalln(err)
			}
			code := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitCode
			os.Exit(int(code))
		case err = <-wait:
			if err != nil {
				log.Println(err)
			}
			lastExitCode = cmd.ProcessState.Sys().(syscall.WaitStatus).ExitCode
		}
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
