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
	chainCoreExe = `C:/Program Files (x86)/Chain/cored.exe`

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
	// Set up chain core logging
	cclog := log.New(os.Stdout, "app=core-manager ", log.Ldate|log.Ltime)
	cclog.Println("Please wait while we check Postgres...")

	// Set up postgres logging
	f, err := os.OpenFile(`C:/Program Files (x86)/Chain/postgres.log`, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) // #nosec
	if err != nil {
		cclog.Fatal("Error opening postgres.log file: " + err.Error())
	}
	defer f.Close()
	pglog := log.New(f, "", log.Ldate|log.Ltime)
	pglog.Println("=====PG CONFIG LOG=====")

	var cmd *exec.Cmd
	// Check if the config file exists--not because we care about the config file,
	// but because we want to know if postgres has been initialized already and
	// the presence of a config file is a good indicator. (If we try to configure
	// PG again, the installer will fail.)
	if _, err := os.Stat(pgDataDir); err == nil {
		if _, err = os.Stat(filepath.Join(pgDataDir, "postgresql.conf")); err != nil {
			pglog.Println("data dir is non-empty, but postgres hasn't been configured yet.")
			cclog.Fatal("Postgres data directory is non-empty, but postgres hasn't been configured yet. You may want to delete the data directory and try again.")
		}
	} else {
		// initdb
		cmd = exec.Command(pg+"initdb.exe", "-D", pgDataDir)
		cmd.Stdout = f
		cmd.Stderr = f

		err = cmd.Run()
		if err != nil {
			pglog.Println("could not run initdb: " + err.Error())
			cclog.Fatal("Postgres could not run initdb. Please check postgres.log for more info.")
		}
	}

	// tweak postgres config
	err = rewriteConfig()
	if err != nil {
		pglog.Println("could not set postgres port or listen addresses: " + err.Error())
		cclog.Fatal("Postgres could not be configured with the port or listen addresses. You may want to manually configure postgresql.conf and try again.")
	}

	// run postgres
	pgCmd := exec.Command(pg+"Postgres.exe", "-D", pgDataDir)
	pgCmd.Stdout = f
	pgCmd.Stderr = f

	err = pgCmd.Start()
	if err != nil {
		pglog.Println("could not start Postgres: " + err.Error())
		cclog.Fatal("Postgres could not be started. Please check postgres.log for more info.")

	}

	pgStatus := make(chan error, 1)
	go func() {
		pgStatus <- pgCmd.Wait()
	}()

	// block until postgres is ready--if we try to create users or db before it's running, it will fail
	blockUntilReady(pglog)

	cmd = exec.Command(pg+"createdb.exe",
		"--port", pgPort,
		"--no-password", // don't _prompt_ for a password. a password still must be provided in the env
		dbName,
	)
	cmd.Stdout = f
	cmd.Stderr = f
	err = cmd.Run()
	if err != nil {
		// it's possible this failed because the database exists already, so don't fail
		// TODO(tessr): investigate using TeeReader (https://godoc.org/io#TeeReader) to read and write the command output
		pglog.Println("could not run createdb: " + err.Error())
		cclog.Printf("WARNING: Postgres did not create database `%s`. It's possible that `%s` exists already. Please check postgres.log for more info.", dbName, dbName)
	}

	pglog.Println("Postgres configured. About to start chain core")
	env := []string{`DATABASE_URL=postgres://localhost:1998/core?sslmode=disable`}
	ccCmd := exec.Command(chainCoreExe)
	ccCmd.Env = mergeEnvLists(os.Environ(), env)
	ccCmd.Stdout = os.Stdout
	ccCmd.Stderr = os.Stderr
	err = ccCmd.Start()
	if err != nil {
		cclog.Println(err)
	}

	ccStatus := make(chan error, 1)
	go func() {
		ccStatus <- ccCmd.Wait()
	}()

	// wait a second for chain core to start,
	// and then navigate to localhost:1999 in the user's browser of choice
	time.Sleep(time.Second)
	cmd = exec.Command("cmd", "/c", "start", "http://localhost:1999")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	if err != nil {
		cclog.Printf("could not open localhost:1999: %s", err)
	}

	var msg = "exit status 0"
	select {
	case pgErr := <-pgStatus:
		if pgErr != nil {
			msg = pgErr.Error()
		}
		cclog.Printf("Postgres died with %s; killing Chain Core", msg)
		ccCmd.Process.Kill()
	case ccErr := <-ccStatus:
		if ccErr != nil {
			msg = ccErr.Error()
		}
		pglog.Printf("Chain Core died with %s; kill Postgres", msg)
		pgCmd.Process.Kill()
	}
}

// append to the config file
func rewriteConfig() error {
	c := pgDataDir + "/postgresql.conf"
	f, err := os.OpenFile(c, os.O_APPEND, 0666) // #nosec
	if err != nil {
		return errors.New("could not open postgresql.conf: " + err.Error())
	}
	defer f.Close()

	_, err = f.WriteString("listen_addresses = '*'    # what IP address(es) to listen on;\n")
	if err != nil {
		return errors.New("could not write listen addresses: " + err.Error())
	}

	_, err = f.WriteString(fmt.Sprintf("port = %s     # (change requires restart)", pgPort))
	if err != nil {
		return errors.New("could not write port: " + err.Error())
	}
	return nil
}

func blockUntilReady(pglog *log.Logger) {
	// TODO(tessr): add a timeout or something so we can't block indefinitely
	for {
		out, err := exec.Command(pg+"pg_isready.exe", "-p", pgPort, "-d", "postgres").Output()
		if err != nil {
			pglog.Printf("out: %s; err: %s", out, err)
		}

		if strings.Contains(string(out), "accepting") {
			return
		}

		pglog.Printf("still waiting for postgres ready status: %s", out)
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
