package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"chain/crypto/ed25519"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "gen":
		_, prv, err := ed25519.GenerateKey(nil)
		must(err)
		os.Stdout.Write(prv)

	case "pub":
		prv, err := ioutil.ReadAll(os.Stdin)
		must(err)
		pub := ed25519.PrivateKey(prv).Public().(ed25519.PublicKey)
		os.Stdout.Write([]byte(pub))

	case "sign":
		if len(os.Args) < 3 {
			usage()
		}
		prvhex := os.Args[2]
		prv, err := hex.DecodeString(prvhex)
		must(err)
		msg, err := ioutil.ReadAll(os.Stdin)
		must(err)
		sig := ed25519.Sign(ed25519.PrivateKey(prv), msg)
		os.Stdout.Write(sig)

	case "verify":
		args := os.Args[2:]
		if len(args) < 1 {
			usage()
		}
		var silent bool
		if args[0] == "-s" {
			silent = true
			args = args[1:]
		}
		if len(args) < 2 {
			usage()
		}
		pub, err := hex.DecodeString(args[0])
		must(err)
		sig, err := hex.DecodeString(args[1])
		must(err)
		msg, err := ioutil.ReadAll(os.Stdin)
		must(err)
		ok := ed25519.Verify(ed25519.PublicKey(pub), msg, sig)
		if !silent {
			if ok {
				fmt.Println("OK")
			} else {
				fmt.Println("BAD")
			}
		}
		if !ok {
			os.Exit(1)
		}

	default:
		usage()
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func usage() {
	opts := []string{
		"gen >privatekey",
		"pub <privatekey >publickey",
		"sign PRIVHEX <message >signature",
		"verify [-s] PRIVHEX SIGHEX <message",
	}
	fmt.Println("Usage:")
	for _, o := range opts {
		fmt.Printf("\t%s %s\n", os.Args[0], o)
	}
	fmt.Println("PRIVHEX is a hex-encoded private key. SIGHEX is a hex-encoded signature.")
	fmt.Println("The verify subcommand prints OK or BAD to stdout;")
	fmt.Println("or, if -s (\"silent\") is given, exits with a zero or non-zero exit code.")
	os.Exit(1)
}
