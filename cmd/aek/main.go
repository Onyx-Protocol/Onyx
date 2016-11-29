package main

import (
	"io"
	"os"

	"chain-stealth/crypto/ca"
)

func main() {
	var rek ca.RecordKey

	_, err := io.ReadFull(os.Stdin, rek[:])
	must(err)
	iek := ca.DeriveIntermediateKey(rek)
	aek := ca.DeriveAssetKey(iek)
	_, err = os.Stdout.Write(aek[:])
	must(err)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
