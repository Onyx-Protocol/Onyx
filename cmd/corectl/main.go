// Command corectl provides miscellaneous control functions for a Chain Core.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"chain/core"
	"chain/core/accesstoken"
	"chain/core/config"
	"chain/core/rpc"
	"chain/crypto/ed25519"
	"chain/env"
	"chain/errors"
	"chain/generated/rev"
	"chain/log"
	"chain/protocol/bc"
)

// config vars
var (
	home    = config.HomeDirFromEnvironment()
	coreURL = env.String("CORE_URL", "http://localhost:1999")

	// build vars; initialized by the linker
	buildTag    = "?"
	buildCommit = "?"
	buildDate   = "?"
)

// We collect log output in this buffer,
// and display it only when there's an error.
var logbuf bytes.Buffer

type command struct {
	f func(*rpc.Client, []string)
}

type grantReq struct {
	Policy    string      `json:"policy"`
	GuardType string      `json:"guard_type"`
	GuardData interface{} `json:"guard_data"`
}

var commands = map[string]*command{
	"config-generator":     {configGenerator},
	"create-block-keypair": {createBlockKeyPair},
	"create-token":         {createToken},
	"config":               {configNongenerator},
	"reset":                {reset},
	"grant":                {grant},
	"revoke":               {revoke},
	"join":                 {joinCluster},
	"init":                 {initCluster},
	"evict":                {evictNode},
	"allow-address":        {allowRaftMember},
	"get":                  {get},
	"add":                  {add},
	"rm":                   {rm},
	"set":                  {set},
	"wait":                 {wait},
}

func main() {
	log.SetOutput(&logbuf)
	env.Parse()

	if len(os.Args) >= 2 && os.Args[1] == "-version" {
		var version string
		if buildTag != "?" {
			// build tag with chain-core-server- prefix indicates official release
			version = strings.TrimPrefix(buildTag, "chain-core-server-")
		} else {
			// version of the form rev123 indicates non-release build
			version = rev.ID
		}
		fmt.Printf("corectl (Chain Core) %s\n", version)
		fmt.Printf("build-commit: %v\n", buildCommit)
		fmt.Printf("build-date: %v\n", buildDate)
		return
	}

	if len(os.Args) < 2 {
		help(os.Stdout)
		os.Exit(0)
	}
	cmd := commands[os.Args[1]]
	if cmd == nil {
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		help(os.Stderr)
		os.Exit(1)
	}
	cmd.f(mustRPCClient(), os.Args[2:])
}

func configGenerator(client *rpc.Client, args []string) {
	const usage = "usage: corectl config-generator [flags] [quorum] [pubkey url]..."
	var (
		quorum  uint32
		signers []*config.BlockSigner
		err     error
	)

	var flags flag.FlagSet
	maxIssuanceWindow := flags.Duration("w", 24*time.Hour, "the maximum issuance window `duration` for this generator")
	flagK := flags.String("k", "", "local `pubkey` for signing blocks")

	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()

	if len(args) == 0 {
		if *flagK != "" {
			quorum = 1
		}
	} else if len(args)%2 != 1 {
		fatalln(usage)
	} else {
		q64, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fatalln(usage)
		}
		quorum = uint32(q64)

		for i := 1; i < len(args); i += 2 {
			pubkey, err := hex.DecodeString(args[i])
			if err != nil {
				fatalln(usage)
			}
			if len(pubkey) != ed25519.PublicKeySize {
				fatalln("error:", "bad ed25519 public key length")
			}
			url := args[i+1]
			signers = append(signers, &config.BlockSigner{
				Pubkey: pubkey,
				Url:    url,
			})
		}
	}

	var blockPub []byte
	if *flagK != "" {
		blockPub, err = hex.DecodeString(*flagK)
		if err != nil {
			fatalln("error: unable to decode block pub")
		}
	}

	conf := &config.Config{
		IsGenerator:         true,
		Quorum:              quorum,
		Signers:             signers,
		MaxIssuanceWindowMs: bc.DurationMillis(*maxIssuanceWindow),
		IsSigner:            *flagK != "",
		BlockPub:            blockPub,
	}

	err = client.Call(context.Background(), "/configure", conf, nil)
	dieOnRPCError(err)

	wait(client, nil)
	var r map[string]interface{}
	err = client.Call(context.Background(), "/info", nil, &r)
	dieOnRPCError(err)
	fmt.Println(r["blockchain_id"])
}

func createBlockKeyPair(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}
	pub := struct {
		Pub ed25519.PublicKey
	}{}
	err := client.Call(context.Background(), "/mockhsm/create-block-key", nil, &pub)
	dieOnRPCError(err)
	fmt.Printf("%x\n", pub.Pub)
}

func createToken(client *rpc.Client, args []string) {
	const usage = "usage: corectl create-token [-net] [name] [policy]"
	var flags flag.FlagSet
	flagNet := flags.Bool("net", false, "DEPRECATED. create a network token instead of client")
	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) == 2 && *flagNet || len(args) < 1 || len(args) > 2 {
		fatalln(usage)
	}

	req := struct{ ID string }{args[0]}
	var tok accesstoken.Token
	// TODO(kr): find a way to make this atomic with the grant below
	err := client.Call(context.Background(), "/create-access-token", req, &tok)
	dieOnRPCError(err)
	fmt.Println(tok.Token)

	grant := grantReq{
		GuardType: "access_token",
		GuardData: map[string]string{"id": tok.ID},
	}
	switch {
	case len(args) == 2:
		grant.Policy = args[1]
	case *flagNet:
		grant.Policy = "crosscore"
		fmt.Fprintln(os.Stderr, "warning: the network flag is deprecated")
	default:
		grant.Policy = "client-readwrite"
		fmt.Fprintln(os.Stderr, "warning: implicit policy name is deprecated")
	}
	err = client.Call(context.Background(), "/create-authorization-grant", grant, nil)
	dieOnRPCError(err, "Auth grant error:")
}

func configNongenerator(client *rpc.Client, args []string) {
	const usage = "usage: corectl config [flags] [blockchain-id] [generator-url]"
	var flags flag.FlagSet
	flagT := flags.String("t", "", "generator access `token`")
	flagK := flags.String("k", "", "local `pubkey` for signing blocks")

	flags.Usage = func() {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) < 2 {
		fatalln(usage)
	}

	var blockchainID bc.Hash
	err := blockchainID.UnmarshalText([]byte(args[0]))
	if err != nil {
		fatalln("error: invalid blockchain ID:", err)
	}

	var blockPub []byte
	if *flagK != "" {
		blockPub, err = hex.DecodeString(*flagK)
		if err != nil {
			fatalln("error: unable to decode block pub")
		}
	}

	var conf config.Config
	conf.BlockchainId = &blockchainID
	conf.GeneratorUrl = args[1]
	conf.GeneratorAccessToken = *flagT
	conf.IsSigner = *flagK != ""
	conf.BlockPub = blockPub

	client.BlockchainID = blockchainID.String()
	err = client.Call(context.Background(), "/configure", conf, nil)
	dieOnRPCError(err)
}

// reset will attempt a reset rpc call on a remote core. If the
// core is not configured with reset capabilities an error is returned.
func reset(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: reset takes no args")
	}

	req := map[string]bool{
		"Everything": true,
	}

	err := client.Call(context.Background(), "/reset", req, nil)
	dieOnRPCError(err)
}

func grant(client *rpc.Client, args []string) {
	editAuthz(client, args, "grant")
}

func revoke(client *rpc.Client, args []string) {
	editAuthz(client, args, "revoke")
}

func editAuthz(client *rpc.Client, args []string, action string) {
	usage := "usage: corectl " + action + " [policy] [guard]"
	var flags flag.FlagSet

	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		fmt.Fprintln(os.Stderr, `
Where guard is one of:
  token=[id]   to affect an access token
  CN=[name]    to affect an X.509 Common Name
  OU=[name]    to affect an X.509 Organizational Unit

The type of guard (before the = sign) is case-insensitive.
`)
		os.Exit(1)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) != 2 {
		fatalln(usage)
	}

	req := grantReq{Policy: args[0]}

	switch typ, data := splitAfter2(args[1], "="); strings.ToUpper(typ) {
	case "TOKEN=":
		req.GuardType = "access_token"
		req.GuardData = map[string]interface{}{"id": data}
	case "CN=":
		req.GuardType = "x509"
		req.GuardData = map[string]interface{}{"subject": map[string]string{"CN": data}}
	case "OU=":
		req.GuardType = "x509"
		req.GuardData = map[string]interface{}{"subject": map[string]string{"OU": data}}
	default:
		fmt.Fprintln(os.Stderr, "unknown guard type", typ)
		fatalln(usage)
	}

	path := map[string]string{
		"grant":  "/create-authorization-grant",
		"revoke": "/delete-authorization-grant",
	}[action]
	err := client.Call(context.Background(), path, req, nil)
	dieOnRPCError(err)
}

// allowRaftMember takes an address and adds it to the list of addresses that are
// allowed for raft cluster members.
func allowRaftMember(client *rpc.Client, args []string) {
	usage := "usage: corectl allow-address [member address]"
	if len(args) != 1 {
		fatalln(usage)
	}

	req := map[string]string{
		"addr": args[0],
	}

	err := client.Call(context.Background(), "/add-allowed-member", req, nil)
	dieOnRPCError(err)
}

// initCluster initializes a new Chain Core cluster.
func initCluster(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: init takes no args")
	}
	err := client.Call(context.Background(), "/init-cluster", nil, nil)
	dieOnRPCError(err)
}

// joinCluster connects to an existing Chain Core cluster at the
// provided boot address.
func joinCluster(client *rpc.Client, args []string) {
	const usage = "usage: corectl join [boot address]"
	if len(args) != 1 {
		fatalln(usage)
	}

	req := map[string]string{"boot_address": args[0]}
	err := client.Call(context.Background(), "/join-cluster", req, nil)
	dieOnRPCError(err)
}

// evictNode evicts a Chain Core cored process from the cluster.
// It does not modify the allowed-member list.
func evictNode(client *rpc.Client, args []string) {
	const usage = "usage: corectl evict [node address]"
	if len(args) != 1 {
		fatalln(usage)
	}

	req := map[string]string{"node_address": args[0]}
	err := client.Call(context.Background(), "/evict", req, nil)
	dieOnRPCError(err)
}

func get(client *rpc.Client, args []string) {
	const usage = "usage: corectl get [key]"
	if len(args) != 1 {
		fatalln(usage)
	}

	req := map[string]interface{}{
		"keys": []interface{}{args[0]},
	}

	var resp map[string][][]string
	err := client.Call(context.Background(), "/config", req, &resp)
	dieOnRPCError(err)
	for _, tuples := range resp {
		for _, tup := range tuples {
			fmt.Println(strings.Join(tup, " "))
		}
	}
}

func add(client *rpc.Client, args []string) {
	const usage = "usage: corectl add [-u] [key] [value]..."

	op := "add"
	if len(args) > 0 && args[0] == "-u" {
		op = "add-or-update"
		args = args[1:]
	}
	if len(args) < 2 {
		fatalln(usage)
	}

	req := map[string]interface{}{
		"updates": []interface{}{
			map[string]interface{}{
				"op":    op,
				"key":   args[0],
				"tuple": args[1:],
			},
		},
	}
	err := client.Call(context.Background(), "/configure", req, nil)
	dieOnRPCError(err)
}

func rm(client *rpc.Client, args []string) {
	const usage = "usage: corectl rm [key] [value]..."
	if len(args) < 2 {
		fatalln(usage)
	}

	req := map[string]interface{}{
		"updates": []interface{}{
			map[string]interface{}{
				"op":    "rm",
				"key":   args[0],
				"tuple": args[1:],
			},
		},
	}
	err := client.Call(context.Background(), "/configure", req, nil)
	dieOnRPCError(err)
}

func set(client *rpc.Client, args []string) {
	const usage = "usage: corectl set [key] [value]..."
	if len(args) < 2 {
		fatalln(usage)
	}

	req := map[string]interface{}{
		"updates": []interface{}{
			map[string]interface{}{
				"op":    "set",
				"key":   args[0],
				"tuple": args[1:],
			},
		},
	}
	err := client.Call(context.Background(), "/configure", req, nil)
	dieOnRPCError(err)
}

func wait(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: wait takes no args")
	}

	for {
		err := client.Call(context.Background(), "/info", nil, nil)
		if err == nil {
			break
		}

		if statusErr, ok := errors.Root(err).(rpc.ErrStatusCode); ok && statusErr.StatusCode/100 != 5 {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func mustRPCClient() *rpc.Client {
	// TODO(kr): refactor some of this cert-loading logic into chain/core
	// and use it from cored as well.
	// Note that this function, unlike maybeUseTLS in cored,
	// does not load the cert and key from env vars,
	// only from the filesystem.
	certFile := filepath.Join(home, "tls.crt")
	keyFile := filepath.Join(home, "tls.key")
	config, err := core.TLSConfig(certFile, keyFile, "")
	if err == core.ErrNoTLS {
		return &rpc.Client{BaseURL: *coreURL}
	} else if err != nil {
		fatalln("error: loading TLS cert:", err)
	}

	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSClientConfig:       config,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	url := *coreURL
	if strings.HasPrefix(url, "http:") {
		url = "https:" + url[5:]
	}

	return &rpc.Client{
		BaseURL: url,
		Client:  &http.Client{Transport: t},
	}
}

func fatalln(v ...interface{}) {
	io.Copy(os.Stderr, &logbuf)
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(2)
}

func dieOnRPCError(err error, prefixes ...interface{}) {
	if err == nil {
		return
	}

	io.Copy(os.Stderr, &logbuf)

	if len(prefixes) > 0 {
		fmt.Fprintln(os.Stderr, prefixes...)
	}

	if msgErr, ok := errors.Root(err).(rpc.ErrStatusCode); ok && msgErr.ErrorData != nil {
		fmt.Fprintln(os.Stderr, "RPC error:", msgErr.ErrorData.ChainCode, msgErr.ErrorData.Message)
		if msgErr.ErrorData.Detail != "" {
			fmt.Fprintln(os.Stderr, "Detail:", msgErr.ErrorData.Detail)
		}
	} else {
		fmt.Fprintln(os.Stderr, "RPC error:", err)
	}

	os.Exit(2)
}

func help(w io.Writer) {
	fmt.Fprintln(w, "usage: corectl [-version] [command] [arguments]")
	fmt.Fprint(w, "\nThe commands are:\n\n")
	for name := range commands {
		fmt.Fprintln(w, "\t", name)
	}
	fmt.Fprint(w, "\nFlags:\n")
	fmt.Fprintln(w, "\t-version   print version information")
	fmt.Fprintln(w)
}

// splitAfter2 is like strings.SplitAfterN with n=2.
// If sep is not in s, it returns a="" and b=s.
func splitAfter2(s, sep string) (a, b string) {
	i := strings.Index(s, sep)
	k := i + len(sep)
	return s[:k], s[k:]
}
