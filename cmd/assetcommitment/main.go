package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"chain-stealth/crypto/ca"
)

type pair struct {
	H ca.AssetCommitment
	C ca.Scalar
}

func main() {
	keyFlag := flag.String("key", "", "hex-encoded asset encryption key")
	flag.Parse()

	var (
		H ca.AssetCommitment
		c ca.Scalar
	)

	if *keyFlag == "" {
		// create a nonblinded asset commitment from a bare assetID
		var assetID ca.AssetID
		_, err := io.ReadFull(os.Stdin, assetID[:])
		must(err)
		H = ca.CreateNonblindedAssetCommitment(assetID)
		c = ca.ZeroScalar
	} else {
		// create a blinded asset commitment from a previous commitment, cumulative blinding factor, and key
		aekBytes, err := hex.DecodeString(*keyFlag)
		must(err)
		var aek ca.AssetKey
		if len(aekBytes) != len(aek) {
			panic(fmt.Errorf("asset encryption key has wrong length %d", len(aekBytes)))
		}
		copy(aek[:], aekBytes)
		inp, err := ioutil.ReadAll(os.Stdin)
		must(err)
		var p pair
		err = json.Unmarshal(inp, &p)
		must(err)
		H, c = ca.CreateBlindedAssetCommitment(p.H, p.C, aek)
	}
	out, err := json.Marshal(pair{H: H, C: c})
	_, err = os.Stdout.Write(out)
	must(err)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
