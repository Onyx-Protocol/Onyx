package raft

import (
	"bytes"
	"chain/net"
	"context"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var idCases = []struct {
	id   uint64
	data []byte
}{
	{1, []byte{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31, 0x89}},
	{2, []byte{0, 0, 0, 0, 0, 0, 0, 2, 0x6d, 0x13, 0xc2, 0x7d}},
}

func TestWriteID(t *testing.T) {
	dir, err := ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}

	sv := Service{state: newTestState()}
	for _, test := range idCases {
		err = sv.writeID(dir, test.id)
		if err != nil {
			t.Error(err)
			continue
		}
		got, err := ioutil.ReadFile(filepath.Join(dir, "id"))
		if err != nil {
			t.Error(err)
			continue
		}
		if !bytes.Equal(got, test.data) {
			t.Errorf("writeID(%d) => %x want %x", test.id, got, test.data)
		}
	}
}

func TestReadID(t *testing.T) {
	sv := Service{state: newTestState()}
	dir, err := ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range idCases {
		err = ioutil.WriteFile(filepath.Join(dir, "id"), test.data, 0666)
		if err != nil {
			t.Error(err)
			continue
		}

		got, err := sv.readID(dir)
		if err != nil {
			t.Error(err)
			continue
		}
		if got != test.id {
			t.Errorf("readID() => %d want %d", got, test.id)
		}
	}
}

var idErrorCases = [][]byte{
	{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31, 0x89, 0}, //add extra byte
	{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31},          //missing byte
	{0, 0, 0, 0, 0, 0, 0, 1, 0x7e, 0x43, 0x31, 0x0},     //bad crc
	{0, 0, 0, 0, 0, 0, 0, 0, 0x8c, 0x28, 0xb2, 0x8a},    //bad id
}

func TestReadIDError(t *testing.T) {
	sv := Service{state: newTestState()}
	dir, err := ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range idErrorCases {
		err = ioutil.WriteFile(filepath.Join(dir, "id"), test, 0666)
		if err != nil {
			t.Error(err)
			continue
		}

		_, err := sv.readID(dir)
		if err == nil {
			t.Errorf("readID of %v => err = nil, want error", test)
		}
	}
}

func TestStartUninitialized(t *testing.T) {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	sv, err := Start("", dir, http.DefaultClient, newTestState())
	if err != nil {
		t.Fatal(err)
	}
	if sv.initialized() {
		t.Error("expected Service.initialized to be false")
	}
	err = sv.WaitRead(ctx)
	if err != ErrUninitialized {
		t.Errorf("sv.WaitRead() = %s, want %s", err, ErrUninitialized)
	}
	_, err = sv.Exec(ctx, []byte{})
	if err != ErrUninitialized {
		t.Errorf("sv.Exec() = %s, want %s", err, ErrUninitialized)
	}
}

func TestClusterSetup(t *testing.T) {
	ctx := context.Background()

	// Create new test cluster
	nodeA, nodeB, nodeC := newTestCluster(ctx, t)
	defer nodeA.cleanup()
	defer nodeB.cleanup()
	defer nodeC.cleanup()

	// Try setting a value on nodeB.
	_, err := nodeB.service.Exec(ctx, set("/foo", "bar"))
	must(t, err)

	// Try reading the value on nodeC's state.
	must(t, nodeC.service.WaitRead(ctx))
	got := nodeC.state.Data["/foo"]
	if got != "bar" {
		t.Errorf("reading /foo, nodeC got %q want %q", got, "bar")
	}
}

func TestNodeEviction(t *testing.T) {
	ctx := context.Background()

	// Create new test cluster
	nodeA, nodeB, nodeC := newTestCluster(ctx, t)
	defer nodeA.cleanup()
	defer nodeB.cleanup()
	defer nodeC.cleanup()
	addrB := nodeB.addr

	// Have nodeA evict nodeB
	nodeA.service.Evict(ctx, addrB)
	must(t, nodeA.service.WaitRead(ctx))
	peers := nodeA.service.state.Peers()
	for _, addr := range peers {
		if addr == addrB {
			t.Errorf("expected nodeB to be evicted: still in peer list")
		}
	}
}

func TestEvictMultiple(t *testing.T) {
	ctx := context.Background()

	// Create new test cluster
	nodeA, nodeB, nodeC := newTestCluster(ctx, t)
	defer nodeA.cleanup()
	defer nodeB.cleanup()
	defer nodeC.cleanup()
	addrB := nodeB.addr
	addrC := nodeC.addr

	// Have nodeA evict nodeB and nodeC
	nodeA.service.Evict(ctx, addrB)
	nodeA.service.Evict(ctx, addrC)
	must(t, nodeA.service.WaitRead(ctx))
	peers := nodeA.service.state.Peers()
	for _, addr := range peers {
		if addr == addrB || addr == addrC {
			t.Errorf("expected node to be evicted: stil in peer list")
		}
	}
}

func TestLeaderEviction(t *testing.T) {
	ctx := context.Background()

	// Create new test cluster
	nodeA, nodeB, nodeC := newTestCluster(ctx, t)
	defer nodeA.cleanup()
	defer nodeB.cleanup()
	defer nodeC.cleanup()
	addrA := nodeA.addr

	// Have nodeC evict nodeA
	nodeC.service.Evict(ctx, addrA)
	must(t, nodeC.service.WaitRead(ctx))
	peers := nodeC.service.state.Peers()
	for _, addr := range peers {
		if addr == addrA {
			t.Errorf("expected node to be evicted: stil in peer list")
		}
	}
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

type testNode struct {
	dir     string
	addr    string
	server  *httptest.Server
	state   *state
	service *Service
}

func (n *testNode) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	n.service.ServeHTTP(rw, req)
}

func (n *testNode) cleanup() {
	n.server.Close()
	n.service.Stop()
	os.RemoveAll(n.dir)
}

// Initializes a new test cluster with three raft nodes
// After calling, must remember to call node.cleanup() on each test node
func newTestCluster(ctx context.Context, t *testing.T) (*testNode, *testNode, *testNode) {
	// Create three uninitialized raft services.
	nodeA := newTestNode(t)
	nodeB := newTestNode(t)
	nodeC := newTestNode(t)

	// Initialize A, creating a fresh cluster.
	must(t, nodeA.service.Init())

	// Update the cluster to allow A, B and C's addresses.
	var err error
	_, err = nodeA.service.Exec(ctx, set("/allowed/"+nodeA.addr, "yes"))
	must(t, err)
	_, err = nodeA.service.Exec(ctx, set("/allowed/"+nodeB.addr, "yes"))
	must(t, err)
	_, err = nodeA.service.Exec(ctx, set("/allowed/"+nodeC.addr, "yes"))
	must(t, err)

	// Add B and C to the cluster.
	must(t, nodeB.service.Join("https://"+nodeA.addr))
	must(t, nodeC.service.Join("https://"+nodeA.addr))
	return nodeA, nodeB, nodeC
}

// newTestNode creates a new local raft Service listening on a random
// port on localhost. It uses the stdlib's httptest package's localhost
// TLS certificates in both its server and client tls configs. It uses
// a simple test kv store implementation for the raft Service's state.
//
// When finished with a node, call its cleanup method to stop the server
// and remove its data directory.
func newTestNode(t *testing.T) *testNode {
	node := new(testNode)

	var err error
	node.dir, err = ioutil.TempDir("", "raft_test.go")
	if err != nil {
		t.Fatal(err)
	}

	// Create a tls server first so that we can retrieve the address
	// and tls certificates to pass in to raft.Start.
	// Note that node.server should not accept incoming requests until
	// node.service is populated, otherwise node.ServeHTTP will panic.
	node.server = httptest.NewTLSServer(node)
	node.addr = node.server.Listener.Addr().String()
	node.state = newTestState()

	// TODO(jackson): In Go 1.9+ use ts.Client()?:
	cert, err := x509.ParseCertificate(node.server.TLS.Certificates[0].Certificate[0])
	if err != nil {
		node.cleanup()
		t.Fatal(err)
	}
	tlsConfig := net.DefaultTLSConfig()
	tlsConfig.RootCAs = x509.NewCertPool()
	tlsConfig.RootCAs.AddCert(cert)
	tlsConfig.Certificates = node.server.TLS.Certificates
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Create the raft service, passing in the server's address
	node.service, err = Start(node.addr, node.dir, httpClient, node.state)
	if err != nil {
		node.cleanup()
		t.Fatal(err)
	}
	return node
}
