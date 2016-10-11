// debugging tool to access the SEE client directly
// usage: xprvseetool [args]
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"chain/crypto/hsm/thales/see"
	"chain/crypto/hsm/thales/xprvseeclient"
)

var dummySighash [32]byte

var (
	trace    = flag.Bool("t", false, "trace")
	ident    = flag.String("i", "dbgxprv1", "key ident")
	userdata = flag.String("u", os.Getenv("CHAIN")+"/crypto/hsm/thales/xprvseemodule/userdata.sar", "userdata")
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
	}

	seeConn, err := see.Open(*userdata)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := recover(); err != nil {
			log.Println("error:", err)
			type hinter interface {
				Hint() string
			}
			if h, ok := err.(hinter); ok {
				log.Println("hint:", h.Hint())
			}
			seeConn.DumpTrace(os.Stderr)
			panic(err)
		} else if *trace {
			seeConn.DumpTrace(os.Stderr)
		}
	}()

	client := xprvseeclient.New(seeConn)

	switch flag.Arg(0) {
	case "gen":
		if flag.NArg() != 2 {
			usage()
		}

		seeinteg, err := seeConn.LoadKey("seeinteg", flag.Arg(1))
		if err != nil {
			panic(err)
		}
		xprvID, err := seeConn.GenerateKey(seeinteg)
		if err != nil {
			panic(err)
		}

		kd, err := client.LoadXprv(xprvID)
		if err != nil {
			panic(err)
		}

		xpub, err := client.DeriveXpub(kd, nil) // root xpub
		if err != nil {
			panic(err)
		}

		err = seeConn.SaveKeyBlobs(xprvID, "custom", *ident)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%x\n", xpub)

	case "derive":
		xprvid, err := seeConn.LoadKey("custom", *ident)
		if err != nil {
			panic(err)
		}
		kd, err := client.LoadXprv(xprvid)
		if err != nil {
			panic(err)
		}

		path := argpath(flag.Args()[1:])
		xpub, err := client.DeriveXpub(kd, path)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%x\n", xpub)

	case "sign":
		xprvid, err := seeConn.LoadKey("custom", *ident)
		if err != nil {
			panic(err)
		}

		kd, err := client.LoadXprv(xprvid)
		if err != nil {
			panic(err)
		}

		path := argpath(flag.Args()[1 : flag.NArg()-1])

		res, err := client.XSign(kd, path, dummySighash)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%x\n", res)

	default:
		log.Println("unknown command")
		usage()
	}
}

func argpath(args []string) [][]byte {
	path := make([][]byte, len(args))
	for i, s := range args {
		p, err := hex.DecodeString(s)
		if err != nil {
			panic(err)
		}
		path[i] = p
	}
	return path
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "\txprvseetool [-t] [-i ident] derive path...")
	fmt.Fprintln(os.Stderr, "\txprvseetool [-t] [-i ident] sign path... msg")
	fmt.Fprintln(os.Stderr, "\txprvseetool [-t] [-i ident] gen userdata-signing-key")
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}
