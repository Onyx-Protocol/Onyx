package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"chain/errors"
)

var (
	flagA = flag.String("a", "api", "`app`")
	flagT = flag.String("t", os.Getenv("USER"), "`target`")
	flagR = flag.String("r", "next", "`release`")
	flagD = flag.Bool("d", false, "delete env vars")
	flagH = flag.Bool("h", false, "show help")

	awsS3 = s3.New(aws.DefaultConfig.WithRegion("us-east-1"))
)

func main() {
	log.SetPrefix("appenv: ")
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] [name|name=value...]\n", os.Args[0])
	}
	flag.Parse()
	if *flagH {
		fmt.Println(strings.TrimSpace(help))
		fmt.Print("\nFlags:\n\n")
		flag.PrintDefaults()
		return
	}
	any, all := containsEquals(flag.Args())
	if !*flagD && flag.NArg() > 1 && !any {
		log.Println("cannot get more than one")
		os.Exit(2)
	} else if !*flagD && any && !all {
		log.Println("cannot mix get and set operations")
		os.Exit(2)
	} else if *flagD && any {
		log.Println("cannot mix delete and set operations")
		os.Exit(2)
	} else if *flagD && flag.NArg() == 0 {
		os.Exit(0) // nothing to do
	}

	path := envPath(*flagA, *flagT, *flagR)
	config, err := getConfig(path)
	if err != nil {
		log.Fatal(err)
	}

	switch args := flag.Args(); {
	case len(args) == 0: // list
		for _, v := range config {
			fmt.Println(v)
		}
	case *flagD:
		if *flagR != "next" {
			log.Println("past releases are readonly")
			log.Println("use '-r next' to delete future values")
			os.Exit(1)
		}
		config = removeEnvNames(config, args)
		sort.Strings(config)
		err = setConfig(path, config)
		if err != nil {
			log.Fatal(err)
		}
	case len(args) == 1 && !strings.Contains(args[0], "="): // print one
		p := args[0] + "="
		for _, kv := range config {
			if strings.HasPrefix(kv, p) {
				fmt.Println(kv[len(p):])
				return
			}
		}
		log.Println("config var not found:", args[0])
		os.Exit(1)
	default: // set
		if *flagR != "next" {
			log.Println("past releases are readonly")
			log.Println("use '-r next' to set future values")
			os.Exit(1)
		}
		config = mergeEnvLists(config, args)
		sort.Strings(config)
		err = setConfig(path, config)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getConfig(path string) (config []string, err error) {
	resp, err := awsS3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("chain-deploy"),
		Key:    &path,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "get %s", path)
	}
	defer resp.Body.Close()

	// NOTE(kr): S3 currently does not support the If-Match HTTP
	// header field on PUT requests, so the etag value returned
	// here isn't useful.
	// According to AWS employee messages from 2006 and 2007,
	// they plan to provide this feature, but they don't know when.
	// See https://forums.aws.amazon.com/thread.jspa?messageID=36162.
	// Until it's available, we run the risk of lost updates.
	err = json.NewDecoder(resp.Body).Decode(&config)
	return config, errors.Wrap(err, "read/decode body")
}

func setConfig(path string, config []string) error {
	b, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}

	_, err = awsS3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String("chain-deploy"),
		Key:    &path,
		Body:   bytes.NewReader(append(b, '\n')),
	})
	return err
}

func containsEquals(a []string) (any, all bool) {
	all = true
	for _, s := range a {
		if strings.Contains(s, "=") {
			any = true
		} else {
			all = false
		}
	}
	return any, all
}

// mergeEnvLists merges the two environment lists such that
// variables with the same name in "in" replace those in "out".
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

// removeEnvNames destructively removes variables
// from env that are listed in names.
func removeEnvNames(env, names []string) []string {
	for i := 0; i < len(env); i++ {
		k := strings.SplitN(env[i], "=", 2)[0]
		if contains(names, k) {
			copy(env[i:], env[i+1:])
			env = env[:len(env)-1]
		}
	}
	return env
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func envPath(app, target, rel string) string {
	if rel == "next" {
		return fmt.Sprintf("/%s/release/%s/env.json", app, target)
	}
	return fmt.Sprintf("/%s/release/%s/%s/env.json", app, target, rel)
}

const help = `
Usage:

	appenv [flags] [name|name=value...]

Command appenv reads and writes Chain app environment
variables (aka config vars).

With no arguments, it prints all config vars and their values.
Given just a name, it prints the value for that config var.
Given one or more name=value arguments, it merges them into
the stack's config.

The special release name "next" refers to config values
that will be used for the next release.
Values can be set only on "next"; past releases are readonly.
`
