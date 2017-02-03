// Command multitool provides miscellaneous Chain-related commands.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/davecgh/go-spew/spew"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

// A timed reader times out its Read() operation after a specified
// time limit.  We use it to wrap os.Stdin in case the user
// unwittingly supplies too few arguments and we block trying to read
// stdin from the terminal.
type timedReader struct {
	io.Reader
	limit time.Duration
}

func (r timedReader) Read(buf []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.limit)
	defer cancel()
	type readResult struct {
		n   int
		err error
	}
	readRes := make(chan readResult)
	go func() {
		n, err := r.Reader.Read(buf)
		readRes <- readResult{n, err}
		close(readRes)
	}()
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case res := <-readRes:
			return res.n, res.err
		}
	}
}

var stdin = timedReader{
	Reader: os.Stdin,
	limit:  5 * time.Second,
}

type command struct {
	fn          func([]string)
	help, usage string
}

var subcommands = map[string]command{
	"assetid":     command{assetid, "compute asset id", "ISSUANCEPROG GENESISHASH ASSETDEFINITIONHASH"},
	"block":       command{block, "decode and pretty-print a block", "BLOCK"},
	"blockheader": command{blockheader, "decode and pretty-print a block header", "BLOCKHEADER"},
	"derive":      command{derive, "derive child from given xpub or xprv and given path", "[-xpub|-xprv] XPUB/XPRV PATH PATH..."},
	"genprv":      command{genprv, "generate prv", ""},
	"genxprv":     command{genxprv, "generate xprv", ""},
	"hex":         command{hexCmd, "string <-> hex", "INPUT"},
	"hmac512":     command{hmac512, "compute the hmac512 digest", "KEY VALUE"},
	"pub":         command{pub, "get pub key from prv, or xpub from xprv", "PRV/XPRV"},
	"script":      command{script, "hex <-> opcodes", "INPUT"},
	"sha3":        command{sha3Cmd, "produce sha3 hash", "INPUT"},
	"sha512":      command{sha512Cmd, "produce sha512 hash", "INPUT"},
	"sha512alt":   command{sha512alt, "produce sha512alt hash", "INPUT"},
	"sign":        command{sign, "sign, using hex PRV or XPRV, the given hex MSG", "PRV/XPRV MSG"},
	"tx":          command{tx, "decode and pretty-print a transaction", "TX"},
	"txhash":      command{txhash, "decode a hex transaction and show its txhash", "TX"},
	"uvarint":     command{uvarint, "decimal <-> hex", "[-from|-to] VAL"},
	"varint":      command{varint, "decimal <-> hex", "[-from|-to] VAL"},
	"verify":      command{verify, "verify, using hex PUB or XPUB and the given hex MSG and SIG", "PUB/XPUB MSG SIG"},
	"zerohash":    command{zerohash, "produce an all-zeroes hash", ""},
}

func init() {
	// This breaks an initialization loop
	subcommands["help"] = command{help, "show help", "[SUBCOMMAND]"}
}

func main() {
	if len(os.Args) < 2 {
		errorf("no subcommand (try \"%s help\")", os.Args[0])
	}
	subcommand := mustSubcommand(os.Args[1])
	subcommand.fn(os.Args[2:])
}

func errorf(msg string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(msg, args...))
	os.Exit(1)
}

func help(args []string) {
	if len(args) > 0 {
		subcommand := mustSubcommand(args[0])
		fmt.Println(subcommand.help)
		fmt.Printf("%s %s\n", args[0], subcommand.usage)
		return
	}

	for name, cmd := range subcommands {
		fmt.Printf("%-16.16s %s\n", name, cmd.help)
	}
}

func mustSubcommand(name string) command {
	if cmd, ok := subcommands[name]; ok {
		return cmd
	}
	errorf("unknown subcommand \"%s\"", name)
	return command{} // not reached
}

func input(args []string, n int, usedStdin bool) (string, bool) {
	if len(args) > n && args[n] != "-" {
		return args[n], usedStdin
	}
	if usedStdin {
		errorf("can use stdin for only one arg")
	}
	b, err := ioutil.ReadAll(stdin)
	if err != nil {
		errorf("unexpected error: %s", err)
	}
	return string(b), true
}

func decodeHex(s string) ([]byte, error) {
	return hex.DecodeString(strings.TrimSpace(s))
}

func mustDecodeHex(s string) []byte {
	res, err := decodeHex(s)
	if err != nil {
		errorf("error decoding hex: %s", err)
	}
	return res
}

func mustDecodeHash(s string) bc.Hash {
	var h bc.Hash
	err := h.UnmarshalText([]byte(strings.TrimSpace(s)))
	if err != nil {
		errorf("error decoding hash: %s", err)
	}
	return h
}

func assetid(args []string) {
	var (
		issuanceInp     string
		initialBlockInp string
		assetdefInp     string
		usedStdin       bool
	)
	issuanceInp, usedStdin = input(args, 0, false)
	initialBlockInp, _ = input(args, 1, usedStdin)
	assetdefInp, _ = input(args, 2, usedStdin)
	issuance := mustDecodeHex(issuanceInp)
	initialBlock := mustDecodeHash(initialBlockInp)
	assetdefHash := bc.EmptyStringHash
	// This case is not supported by multitool yet. Keep this in mind when moving this func to its own command.
	if len(assetdefInp) > 0 {
		assetdefHash = mustDecodeHash(assetdefInp)
	}
	assetID := bc.ComputeAssetID(issuance, initialBlock, 1, assetdefHash)
	fmt.Println(assetID.String())
}

func block(args []string) {
	inp, _ := input(args, 0, false)
	var block bc.Block
	err := json.Unmarshal([]byte(inp), &block)
	if err != nil {
		errorf("error unmarshaling block: %s", err)
	}
	spew.Printf("%v\n", block)
}

func blockheader(args []string) {
	inp, _ := input(args, 0, false)
	var bh bc.BlockHeader
	err := json.Unmarshal([]byte(inp), &bh)
	if err != nil {
		errorf("error unmarshaling blockheader: %s", err)
	}
	spew.Printf("%v\n", bh)
}

func derive(args []string) {
	if len(args) == 0 {
		errorf("must specify -xprv or -xpub, key, and path")
	}
	which := args[0]
	args = args[1:]

	switch which {
	case "-xprv", "-xpub":
		// ok
	default:
		errorf("must specify -xprv or -xpub")
	}

	k, _ := input(args, 0, false)
	path := make([][]byte, 0, len(args)-1)
	for _, a := range args[1:] {
		p, err := hex.DecodeString(a)
		if err != nil {
			errorf("could not parse %s as hex string", a)
		}
		path = append(path, p)
	}

	k = strings.TrimSpace(k)

	if which == "-xprv" {
		var xprv chainkd.XPrv
		err := xprv.UnmarshalText([]byte(k))
		if err != nil {
			errorf("could not parse key")
		}
		derived := xprv.Derive(path)
		fmt.Println(derived.String())
		return
	}

	var xpub chainkd.XPub
	err := xpub.UnmarshalText([]byte(k))
	if err != nil {
		errorf("could not parse key")
	}
	derived := xpub.Derive(path)
	fmt.Println(derived.String())
}

func genprv(_ []string) {
	_, prv, err := ed25519.GenerateKey(nil)
	if err != nil {
		errorf("unexpected error %s", err)
	}
	fmt.Println(hex.EncodeToString(prv))
}

func genxprv(_ []string) {
	xprv, _, err := chainkd.NewXKeys(nil)
	if err != nil {
		errorf("unexpected error %s", err)
	}
	fmt.Println(xprv.String())
}

func hexCmd(args []string) {
	inp, _ := input(args, 0, false)
	b, err := decodeHex(inp)
	if err == nil {
		fmt.Println(string(b))
	} else {
		fmt.Println(hex.EncodeToString([]byte(inp)))
	}
}

func hmac512(args []string) {
	key, usedStdin := input(args, 0, false)
	val, _ := input(args, 1, usedStdin)
	mac := hmac.New(sha512.New, mustDecodeHex(key))
	mac.Write(mustDecodeHex(val))
	fmt.Println(hex.EncodeToString(mac.Sum(nil)))
}

func pub(args []string) {
	inp, _ := input(args, 0, false)
	var xprv chainkd.XPrv
	err := xprv.UnmarshalText([]byte(strings.TrimSpace(inp)))
	if err == nil {
		fmt.Println(xprv.XPub().String())
		return
	}
	prv := ed25519.PrivateKey(mustDecodeHex(inp))
	pub := prv.Public().(ed25519.PublicKey)
	fmt.Println(hex.EncodeToString(pub))
}

func script(args []string) {
	inp, _ := input(args, 0, false)
	b, err := decodeHex(inp)
	if err == nil {
		dis, err := vm.Disassemble(b)
		if err == nil {
			fmt.Println(dis)
			return
		}
		// The input parsed as hex but not as a compiled program. Maybe
		// it's an uncompiled program that just looks like hex. Fall
		// through and try it that way.
	}
	parsed, err := vm.Assemble(inp)
	if err == nil {
		fmt.Println(hex.EncodeToString(parsed))
		return
	}
	errorf("could not parse input")
}

func sha3Cmd(args []string) {
	inp, _ := input(args, 0, false)
	b := mustDecodeHex(inp)
	h := sha3.Sum256(b)
	fmt.Println(hex.EncodeToString(h[:]))
}

func sha512Cmd(args []string) {
	inp, _ := input(args, 0, false)
	b := mustDecodeHex(inp)
	h := sha512.Sum512(b)
	fmt.Println(hex.EncodeToString(h[:]))
}

func sha512alt(args []string) {
	inp, _ := input(args, 0, false)
	b := mustDecodeHex(inp)
	h := sha512.Sum512(b)
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64
	fmt.Println(hex.EncodeToString(h[:32]))
}

func sign(args []string) {
	if len(args) == 0 {
		errorf("must specify -xprv or -prv, plus msg")
	}
	which := args[0]
	args = args[1:]

	switch which {
	case "-xprv", "-prv":
		// ok
	default:
		errorf("must specify -xprv or -prv")
	}

	var (
		keyInp, msgInp string
		usedStdin      bool
	)
	keyInp, usedStdin = input(args, 0, false)
	msgInp, _ = input(args, 1, usedStdin)

	keyInp = strings.TrimSpace(keyInp)
	msg := mustDecodeHex(msgInp)
	var signed []byte

	if which == "-xprv" {
		var xprv chainkd.XPrv
		err := xprv.UnmarshalText([]byte(keyInp))
		if err != nil {
			errorf("could not parse xprv")
		}
		signed = xprv.Sign(msg)
	} else {
		prv := ed25519.PrivateKey(mustDecodeHex(keyInp))
		signed = ed25519.Sign(prv, msg)
	}

	fmt.Println(hex.EncodeToString(signed))
}

func tx(args []string) {
	inp, _ := input(args, 0, false)
	var tx bc.TxData
	err := tx.UnmarshalText([]byte(strings.TrimSpace(inp)))
	if err != nil {
		errorf("error unmarshaling tx: %s", err)
	}
	spew.Printf("%v\n", tx)
}

func txhash(args []string) {
	inp, _ := input(args, 0, false)
	var tx bc.Tx
	err := tx.UnmarshalText([]byte(strings.TrimSpace(inp)))
	if err != nil {
		errorf("error unmarshaling tx: %s", err)
	}
	h := tx.ID
	fmt.Printf("%x\n", h[:])
}

func varint(args []string) {
	dovarint(args, true)
}

func uvarint(args []string) {
	dovarint(args, false)
}

func dovarint(args []string, signed bool) {
	var mode string
	if len(args) > 0 {
		switch args[0] {
		case "-from", "-to":
			mode = args[0]
			args = args[1:]
		}
	}
	val, _ := input(args, 0, false)
	if mode == "" {
		if strings.HasPrefix(val, "0x") {
			mode = "-from"
			val = strings.TrimPrefix(val, "0x")
		} else {
			_, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				mode = "-to"
			} else {
				mode = "-from"
			}
		}
	}
	switch mode {
	case "-to":
		val10, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			errorf("could not parse base 10 int")
		}
		var (
			buf [10]byte
			n   int
		)
		if signed {
			n = binary.PutVarint(buf[:], val10)
		} else {
			n = binary.PutUvarint(buf[:], uint64(val10))
		}
		fmt.Println(hex.EncodeToString(buf[:n]))
	case "-from":
		val16 := mustDecodeHex(val)
		if signed {
			n, nbytes := binary.Varint(val16)
			if nbytes <= 0 {
				errorf("could not parse varint")
			}
			fmt.Println(n)
		} else {
			n, nbytes := binary.Uvarint(val16)
			if nbytes <= 0 {
				errorf("could not parse varint")
			}
			fmt.Println(n)
		}
	}
}

func verify(args []string) {
	var (
		keyInp, msgInp, sigInp string
		usedStdin              bool
	)
	keyInp, usedStdin = input(args, 0, false)
	msgInp, usedStdin = input(args, 1, usedStdin)
	sigInp, _ = input(args, 2, usedStdin)

	keyInp = strings.TrimSpace(keyInp)
	msg := mustDecodeHex(msgInp)
	sig := mustDecodeHex(sigInp)

	var verified bool

	switch len(keyInp) {
	case 128:
		var xpub chainkd.XPub
		err := xpub.UnmarshalText([]byte(keyInp))
		if err != nil {
			errorf("could not parse xpub")
		}
		verified = xpub.Verify(msg, sig)

	case 64:
		pub := ed25519.PublicKey(mustDecodeHex(keyInp))
		verified = ed25519.Verify(pub, msg, sig)

	default:
		errorf("could not parse key")
	}

	if verified {
		fmt.Println("verified")
	} else {
		fmt.Println("not verified")
	}
}

func zerohash(_ []string) {
	fmt.Println(bc.Hash{}.String())
}
