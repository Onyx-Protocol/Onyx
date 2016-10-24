package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func runIn(dir string, c *exec.Cmd) {
	c.Dir = dir
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		panic(fmt.Errorf("%s: %v", strings.Join(c.Args, " "), err))
	}
}

func dirCmd(dir, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd
}

func runOutput(dir string, c *exec.Cmd) []byte {
	c.Dir = dir
	c.Stderr = os.Stderr
	out, err := c.Output()
	if err != nil {
		panic(fmt.Errorf("%s: %v", strings.Join(c.Args, " "), err))
	}
	return out
}

func pipeline(cmds ...*exec.Cmd) ([]byte, error) {
	var last *exec.Cmd
	for _, cmd := range cmds {
		if last != nil {
			in, err := last.StdoutPipe()
			if err != nil {
				return nil, err
			}
			cmd.Stdin = in
		}
		last = cmd
	}

	buf := bytes.NewBuffer(nil)
	last.Stdout = buf

	for _, cmd := range cmds {
		err := cmd.Start()
		if err != nil {
			return nil, err
		}
	}

	for _, cmd := range cmds {
		err := cmd.Wait()
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func catch() {
	if err := recover(); err != nil {
		switch err := err.(type) {
		case error:
			log.Println(err)
			sayf(":frowning:\n%s", err)
		default:
			panic(err)
		}
	}
}

func contains(s string, a []string) bool {
	for _, s1 := range a {
		if s == s1 {
			return true
		}
	}
	return false
}
