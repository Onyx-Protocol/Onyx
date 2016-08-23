package main

import (
	"bufio"
	"encoding/hex"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestCompiler(t *testing.T) {
	dir, err := os.Open("tests")
	if err != nil {
		t.Fatal(err)
	}

	names, err := dir.Readdirnames(0)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range names {
		// TODO: fix these broken tests
		if strings.HasPrefix(name, "auction") || strings.HasPrefix(name, "collateral") || strings.HasPrefix(name, "orderbook") {
			continue
		}

		if strings.HasSuffix(name, ".input") {
			t.Logf("reading %s", name)
			input, err := ioutil.ReadFile("tests/" + name)
			if err != nil {
				t.Fatal(err)
			}
			contracts, err := parse(input)
			if err != nil {
				t.Errorf("parsing %s: %s", name, err)
				continue
			}
			translated, err := translate(contracts[0], contracts)
			if err != nil {
				t.Errorf("translating %s: %s", name, err)
				continue
			}
			bytes, err := translated.getBytes()
			if err != nil {
				t.Errorf("parsing opcodes from %s: %s", name, err)
			}

			prefix := strings.TrimSuffix(name, ".input")
			output, err := os.Open("tests/" + prefix + ".output")
			if err != nil {
				t.Fatal(err)
			}
			func() {
				defer output.Close()
				scanner := bufio.NewScanner(output)
				scanner.Split(bufio.ScanLines)
				var (
					contractHexNext bool
					expectedHex     string
				)
				for scanner.Scan() {
					line := scanner.Text()
					if contractHexNext {
						expectedHex = line
						break
					} else if line == "Contract hex:" {
						contractHexNext = true
					}
				}
				gotHex := hex.EncodeToString(bytes)
				if gotHex != expectedHex {
					t.Errorf("mismatch in %s: got %s, expected %s", name, gotHex, expectedHex)
				}
			}()
		}
	}
}
