package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"chain/cos/txscript"
	"chain/crypto/hash256"
)

var flagDebug = flag.Bool("debug", false, "run in debug mode")

func main() {
	flag.Parse()

	var (
		in  []byte
		err error
	)

	if a := flag.Args(); len(a) > 0 {
		in, err = ioutil.ReadFile(a[0])
	} else {
		in, err = ioutil.ReadAll(os.Stdin)
	}
	if err != nil {
		panic(err)
	}
	contracts, err := parse(in)
	if err != nil {
		panic(err)
	}
	first := true
	for _, contract := range contracts {
		res, err := translate(contract, contracts)
		if err != nil {
			panic(err)
		}
		if !first {
			fmt.Printf("\n")
		}
		fmt.Printf("Contract \"%s\":\n", contract.name)
		var (
			longest int
			allOps  []string
		)
		for _, translation := range res {
			if len(translation.ops) > longest {
				longest = len(translation.ops)
			}
			allOps = append(allOps, translation.ops)
		}
		f := fmt.Sprintf("%%-%d.%ds  # <top> %%s\n", longest, longest)
		var initStack []string
		for _, p := range contract.params {
			initStack = append(initStack, p.name)
		}
		if len(contract.clauses) > 1 {
			initStack = append(initStack, "[clause selector] ...clause args...")
		} else {
			for _, p := range contract.clauses[0].params {
				initStack = append(initStack, p.name)
			}
		}
		fmt.Printf(f, "", strings.Join(initStack, " "))
		for _, translation := range res {
			ops := translation.ops
			stack := translation.stack
			strs := make([]string, 0, len(stack))
			for _, item := range stack {
				strs = append(strs, item.name)
			}
			fmt.Printf(f, ops, strings.Join(strs, " "))
		}

		parsed, err := txscript.ParseScriptString(strings.Join(allOps, " "))
		if err != nil {
			panic(err)
		}

		contractHex := hex.EncodeToString(parsed)

		fmt.Println("\nContract hex:")
		fmt.Println(contractHex)

		hash := hash256.Sum(parsed)

		fmt.Println("\nContracthash hex:")
		fmt.Println(hex.EncodeToString(hash[:]))

		pkscript, err := txscript.PayToContractHash(hash, nil, txscript.ScriptVersion1)
		if err != nil {
			panic(err)
		}

		// Passed nil for params above.  Add in placeholders for them
		// "manually."
		pkscriptPrefix := pkscript[:2] // <scriptVersion> DROP
		var pkscriptSuffix []byte
		if len(contract.params) > 0 {
			pkscriptSuffix = txscript.AddInt64ToScript(nil, int64(len(contract.params)))
			pkscriptSuffix = append(pkscriptSuffix, txscript.OP_ROLL)
		}
		pkscriptSuffix = append(pkscriptSuffix, pkscript[2:]...) // DUP HASH256 <hash> EQUALVERIFY EVAL

		fmt.Println("\nPkscript hex:")
		fmt.Printf("%s", hex.EncodeToString(pkscriptPrefix))
		for n := len(contract.params) - 1; n >= 0; n-- {
			fmt.Printf("<%s>", contract.params[n].name)
		}
		fmt.Printf("%s\n", hex.EncodeToString(pkscriptSuffix))

		for i, clause := range contract.clauses {
			fmt.Printf("\nRedeem %s.%s:\n", contract.name, clause.name)
			for n := len(clause.params) - 1; n >= 0; n-- {
				p := clause.params[n]
				fmt.Printf("<%s>", p.name)
			}
			var redeem []byte
			if len(contract.clauses) > 1 {
				redeem = txscript.AddInt64ToScript(nil, int64(i+1))
			}
			redeem = txscript.AddDataToScript(redeem, parsed)
			fmt.Printf("%s\n", hex.EncodeToString(redeem))
		}
		first = false
	}
}
