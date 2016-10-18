package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// This file is used to initialize and configure a Postgres database that
// Chain Core can use on Windows.
// It can be cross-compiled using
// `GOOS=windows GOARCH=amd64 go build chain/installer/windows/ChainMgr`

const (

	// The Chain Core executable itself.
	chainCoreExe = `C:/Program Files (x86)/Chain/chain-core.exe`

	// Data directory for Postgres to store all of its stuff for a specific db.
	pgDataDir = `C:/Program Files (x86)/Chain/data`

	// Path for all the Postgres binaries.
	pg = `C:/Program Files (x86)/Chain/Postgres/bin/`

	// Port this db will listen on. Also the year I started kindergarten.
	pgPort = "1998"

	// Postgres user ("postgres" is the default). NOT the system user.
	pgUser = "postgres"

	// Password for that postgres user to use.
	pgPassword = "password"

	// Database name. Changing this requires passing the correct environment vars
	// to Chain Core.
	dbName = "core"
)

func main() {
	// Set up logging
	// TODO(tessr): better temp file
	f, err := os.OpenFile(`C:/Program Files (x86)/Chain/install.log`, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("error opening log file: " + err.Error())
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println("=====PG CONFIG LOG=====")

	var cmd *exec.Cmd
	// Check if the config file exists--not because we care about the config file,
	// but because we want to know if postgres has been initialized already and
	// the presence of a config file is a good indicator. (If we try to configure
	// PG again, the installer will fail.)
	if _, err := os.Stat(pgDataDir); err == nil {
		if _, err = os.Stat(filepath.Join(pgDataDir, "postgresql.conf")); err != nil {
			log.Fatal("data dir is non-empty, but postgres hasn't been configured yet.")
		}
	} else {
		// initdb
		cmd = exec.Command(pg+"initdb.exe", "-D", pgDataDir)
		cmd.Stdout = f
		cmd.Stderr = f

		err = cmd.Run()
		if err != nil {
			// TODO(tessr): user-friendly message about manually configuring pg?
			log.Fatal("could not run initdb: " + err.Error())
		}
	}

	// tweak postgres config
	err = rewriteConfig()
	if err != nil {
		log.Fatal("could not set postgres port or listen addresses: " + err.Error())
	}

	// run postgres
	cmd = exec.Command(pg+"Postgres.exe", "-D", pgDataDir)
	cmd.Stdout = f
	cmd.Stderr = f

	err = cmd.Start()
	if err != nil {
		log.Fatal("could not start Postgres: " + err.Error())
	}

	// block until postgres is ready--if we try to create users or db before it's running, it will fail
	blockUntilReady()

	cmd = exec.Command(pg+"createdb.exe",
		"--port", pgPort,
		"--no-password", // don't _prompt_ for a password. a password still must be provided in the env
		dbName,
	)
	cmd.Stdout = f
	cmd.Stderr = f
	err = cmd.Run()
	if err != nil {
		log.Fatal("could not run createdb: " + err.Error())
	}

	log.Println("about to start chain core")
	env := []string{`DATABASE_URL=postgres://localhost:1998/core?sslmode=disable`}
	cmd = exec.Command(chainCoreExe)
	cmd.Env = mergeEnvLists(os.Environ(), env)
	cmd.Stdout = f
	cmd.Stderr = f
	log.Println(cmd.Env)
	err = cmd.Start()
	if err != nil {
		log.Println(err)
	}
}

// append to the config file
func rewriteConfig() error {
	c := pgDataDir + "/postgresql.conf"
	f, err := os.OpenFile(c, os.O_APPEND, 0666)
	if err != nil {
		return errors.New("could not open postgresql.conf: " + err.Error())
	}
	defer f.Close()

	_, err = f.WriteString("listen_addresses = '*'    # what IP address(es) to listen on;")
	if err != nil {
		return errors.New("could not write listen addresses: " + err.Error())
	}

	_, err = f.WriteString(fmt.Sprintf("port = %s     # (change requires restart)", pgPort))
	if err != nil {
		return errors.New("could not write port: " + err.Error())
	}
	return nil
}

func blockUntilReady() {
	// TODO(tessr): add a timeout or something so we can't block indefinitely
	for {
		out, err := exec.Command(pg+"pg_isready.exe", "-p", pgPort, "-d", "postgres").Output()
		if err != nil {
			log.Printf("out: %s; err: %s", out, err)
		}

		if strings.Contains(string(out), "accepting") {
			return
		}

		log.Printf("not ready, got %s", out)
		time.Sleep(500 * time.Millisecond)
	}
}

// mergeEnvLists merges the two environment lists such that
// variables with the same name in "in" replace those in "out".
// Pulled straight outta chain core.
func mergeEnvLists(out, in []string) []string {
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
