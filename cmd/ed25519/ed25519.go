package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/agl/ed25519"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "gen":
		_, prv, err := ed25519.GenerateKey(rand.Reader)
		must(err)
		os.Stdout.Write(prv[:])

	case "pub":
		prv, err := ioutil.ReadAll(os.Stdin)
		must(err)
		if len(prv) != ed25519.PrivateKeySize {
			panic(fmt.Errorf("bad private key size %d", len(prv)))
		}
		// Abstraction violation: would prefer this code didn't know that
		// prv[32:] is the pubkey.
		os.Stdout.Write(prv[32:])

	case "sign":
		if len(os.Args) < 3 {
			usage()
		}
		prvhex := os.Args[2]
		prv, err := hex.DecodeString(prvhex)
		must(err)
		msg, err := ioutil.ReadAll(os.Stdin)
		must(err)
		if len(prv) != ed25519.PrivateKeySize {
			panic(fmt.Errorf("bad private key size %d", len(prv)))
		}
		var prvbuf [ed25519.PrivateKeySize]byte
		copy(prvbuf[:], prv)
		sig := ed25519.Sign(&prvbuf, msg)
		os.Stdout.Write(sig[:])

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
		if len(pub) != ed25519.PublicKeySize {
			panic(fmt.Errorf("bad public key size %d", len(pub)))
		}
		var pubbuf [ed25519.PublicKeySize]byte
		copy(pubbuf[:], pub)
		if len(sig) != ed25519.PublicKeySize {
			panic(fmt.Errorf("bad signature size %d", len(sig)))
		}
		var sigbuf [ed25519.SignatureSize]byte
		copy(sigbuf[:], sig)
		ok := ed25519.Verify(&pubbuf, msg, &sigbuf)
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
