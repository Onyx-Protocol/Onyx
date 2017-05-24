// Package raft provides raft coordination.
package raft

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/snap/snappb"
	"github.com/coreos/etcd/wal"
	"github.com/coreos/etcd/wal/walpb"

	"chain/errors"
	"chain/log"
	"chain/net/http/httperror"
)

// TODO(kr): do we need a "client" mode?
// So we can have many coreds without all of them
// having to be active raft nodes.
// (Raft isn't really meant for more than a handful
// of consensus participants.)

const (
	tickDur           = 100 * time.Millisecond
	electionTick      = 10
	heartbeatTick     = 1
	maxRaftReqSize    = 10e6 // 10MB
	snapCount         = 10000
	dummyWriteTimeout = 50 * time.Millisecond

	nSnapCatchupEntries uint64 = 10000

	contentType = "application/octet-stream"
)

var (
	// ErrExistingCluster is returned from Init or Join when the Service
	// is already connected to a raft cluster.
	ErrExistingCluster = errors.New("already connected to a raft cluster")

	// ErrUninitialized is returned when the Service is not yet connected
	// to any cluster.
	ErrUninitialized = errors.New("no raft cluster configured")

	// ErrAddressNotAllowed is returned from Join when the node's address
	// is not in the provided cluster's allowed address list.
	ErrAddressNotAllowed = errors.New("address is not allowed")

	// ErrPeerUninitialized is returned when a peer node indicates it's
	// not yet initialized.
	ErrPeerUninitialized = errors.New("peer is uninitialized")
)

var (
	errBadRequest  = errors.New("bad request")
	errorFormatter = httperror.Formatter{
		Default:     httperror.Info{500, "CH000", "Chain API Error"},
		IsTemporary: func(info httperror.Info, _ error) bool { return info.ChainCode == "CH000" },
		Errors: map[error]httperror.Info{
			errBadRequest:        {400, "CH003", "Invalid request body"},
			ErrAddressNotAllowed: {400, "CH162", "Address is not allowed"},
			ErrUninitialized:     {400, "CH163", "No cluster configured"},
		},
	}
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

// Service performs raft coordination.
type Service struct {
	// config
	dir     string
	laddr   string
	mux     *http.ServeMux
	rctxReq chan rctxReq
	wctxReq chan wctxReq
	donec   chan struct{}
	client  *http.Client

	// config set during init/join/restart. immutable once set.
	// it is ok to read without keeping startMu locked in
	// code paths where Service is known to be initialized.
	startMu  sync.Mutex
	raftNode raft.Node
	id       uint64

	errMu sync.Mutex
	err   error

	confChangeID uint64 // atomic access only

	// The storage object is purely for internal use
	// by the raft.Node to maintain consistent persistent state.
	// All client code accesses the cluster state via our Service
	// object, which keeps a local, in-memory copy of the
	// complete current state.
	raftStorage *raft.MemoryStorage

	// The actual replicated data set.
	stateMu   sync.Mutex
	stateCond sync.Cond
	state     State
	confState raftpb.ConfState
	done      bool

	// Current log position, accessed only from runUpdates goroutine
	snapIndex uint64
}

type State interface {
	AppliedIndex() uint64
	Apply(data []byte, index uint64) (satisfied bool)
	Snapshot() (data []byte, index uint64, err error)
	RestoreSnapshot(data []byte, index uint64) error
	SetPeerAddr(id uint64, addr string)
	GetPeerAddr(id uint64) (addr string)
	RemovePeerAddr(id uint64)
	IsAllowedMember(addr string) bool
	NextNodeID() (id, version uint64)
	EmptyWrite() (instruction []byte)
	IncrementNextNodeID(oldID uint64, index uint64) (instruction []byte)
}

// rctxReq is a "read context" request.
type rctxReq struct {
	rctx  []byte
	index chan uint64
}

// wctx is a "write context" request.
type wctxReq struct {
	wctx      []byte
	satisfied chan bool
}

type proposal struct {
	Wctx        []byte
	Instruction []byte
}

// nodeJoin is the data used when a new node joins a cluster
type nodeJoin struct {
	ID   uint64
	Snap []byte
}

// Start starts the raft algorithm.
//
// Param laddr is the local address,
// to be used by peers to send messages to the local node.
// The returned *Service handles HTTP requests
// matching the ServeMux pattern /raft/.
// The caller is responsible for registering this handler
// to receive incoming requests on laddr. For example:
//   rs, err := raft.Start(addr, ...)
//   ...
//   http.Handle("/raft/", rs)
//   http.ListenAndServe(addr, nil)
//
// Param dir is the filesystem location for all persistent storage
// for this raft node. If dir exists and is populated, the returned
// Service will be immediately ready for use.
// It has three entries:
//   id    file containing the node's member id (never changes)
//   snap  file containing the last complete state snapshot
//   wal   dir containing the write-ahead log
//
// If dir doesn't exist or is empty, the caller must configure the
// Service before using it by either calling Init to initialize a
// new raft cluster or Join to join an existing raft cluster.
//
// The returned *Service will use httpClient for outbound
// connections to peers.
func Start(laddr, dir string, httpClient *http.Client, state State) (*Service, error) {
	// TODO(tessr): configure raft service using run options
	ctx := context.Background()

	// We advertise laddr as the way for peers to reach this process.
	// Make sure our own TLS cert is valid for our own name.
	// (By convention, we use the same cert as a server and client
	// when acting as a raft peer, so we use our *client* cert here.)
	if err := verifyTLSName(laddr, httpClient); err != nil {
		return nil, errors.Wrap(err, "advertised name does not match TLS cert")
	}
	sv := &Service{
		dir:         dir,
		laddr:       laddr,
		mux:         http.NewServeMux(),
		raftStorage: raft.NewMemoryStorage(),
		state:       state,
		donec:       make(chan struct{}),
		rctxReq:     make(chan rctxReq),
		wctxReq:     make(chan wctxReq),
		client:      httpClient,
	}
	sv.stateCond.L = &sv.stateMu

	// TODO(kr): grpc
	sv.mux.HandleFunc("/raft/join", sv.serveJoin)
	sv.mux.HandleFunc("/raft/msg", sv.serveMsg)

	walobj, err := sv.recover()
	if err != nil {
		return nil, err
	}
	// If there's no WAL, then this is a new node. The caller is responsible
	// for calling either Init to initialize a new cluster or Join to join
	// an existing cluster.
	if walobj == nil {
		return sv, nil
	}

	id, err := readID(sv.dir)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	raftNode := raft.RestartNode(sv.config(id))
	err = raftNode.Campaign(ctx)
	if err != nil {
		log.Error(ctx, err, "election failed") // ok to continue
	}

	// Start the algorithm. It is okay to not lock startMu since
	// sv hasn't escaped yet.
	sv.id = id
	sv.raftNode = raftNode
	sv.startLocked(walobj)

	return sv, nil
}

// startLocked begins the raft algorithm. It requires sv.startMu
// to already be locked.
func (sv *Service) startLocked(walobj *wal.WAL) {
	go sv.runUpdates(walobj)
	go runTicks(sv.raftNode)
}

// initialized returns whether the service's raft cluster is
// initialized. If not initialized, Exec and WaitRead will
// error with ErrUninitialized.
func (sv *Service) initialized() bool {
	sv.startMu.Lock()
	defer sv.startMu.Unlock()
	return sv.raftNode != nil
}

func (sv *Service) config(id uint64) *raft.Config {
	return &raft.Config{
		ID:              id,
		ElectionTick:    electionTick,
		HeartbeatTick:   heartbeatTick,
		Storage:         sv.raftStorage,
		Applied:         sv.state.AppliedIndex(),
		MaxSizePerMsg:   4096,
		MaxInflightMsgs: 256,
		Logger:          &raft.DefaultLogger{Logger: stdlog.New(ioutil.Discard, "", 0)},
	}
}

// Init initializes a new Raft cluster.
func (sv *Service) Init() error {
	const firstNodeID = 1
	ctx := context.Background()

	sv.startMu.Lock()
	defer sv.startMu.Unlock()

	if sv.raftNode != nil {
		return ErrExistingCluster
	}

	log.Printkv(ctx, "raftid", firstNodeID)
	err := writeID(sv.dir, firstNodeID)
	if err != nil {
		return err
	}
	err = os.Remove(sv.walDir())
	if err != nil {
		return errors.Wrap(err)
	}
	walobj, err := wal.Create(sv.walDir(), nil)
	if err != nil {
		return errors.Wrap(err)
	}

	peers := []raft.Peer{{ID: firstNodeID, Context: []byte(sv.laddr)}}
	raftNode := raft.StartNode(sv.config(firstNodeID), peers)

	sv.id = firstNodeID
	sv.raftNode = raftNode

	// StartNode appends to the initial log a ConfChangeAddNode entry for
	// each peer. We can't campaign until these entries are applied,
	// so synchronously apply them before continuing.
	rd := <-raftNode.Ready()
	sv.runUpdatesReady(rd, walobj, map[string]chan bool{})

	sv.startLocked(walobj)

	// campaign immediately to avoid waiting electionTick ticks in tests
	err = raftNode.Campaign(ctx)
	if err != nil {
		log.Error(ctx, err, "election failed") // ok to continue
	}
	return nil
}

// Join connects to an existing Raft cluster.
// bootURL gives the location of an existing cluster
// for the local process to join. It can be either
// the concrete address of any single cluster member
// or it can point to a load balancer for the whole
// cluster, if one exists.
func (sv *Service) Join(bootURL string) error {
	sv.startMu.Lock()
	defer sv.startMu.Unlock()

	if sv.raftNode != nil {
		return ErrExistingCluster
	}

	id, walobj, err := sv.join(sv.laddr, bootURL) // sets state
	if err != nil {
		return err
	}
	sv.id = id
	sv.raftNode = raft.RestartNode(sv.config(id))
	sv.startLocked(walobj)
	return nil
}

// Err returns a serious error preventing this process from
// operating normally or making progress, if any.
// Note that it is possible for a Service to recover automatically
// from some errors returned by Err.
func (sv *Service) Err() error {
	sv.errMu.Lock()
	defer sv.errMu.Unlock()
	return sv.err
}

func (sv *Service) runUpdatesReady(rd raft.Ready, wal *wal.WAL, writers map[string]chan bool) {
	wal.Save(rd.HardState, rd.Entries)
	if !raft.IsEmptySnap(rd.Snapshot) {
		sv.redo(func() error {
			return sv.saveSnapshot(&rd.Snapshot)
		})
		sv.redo(func() error {
			// Note: wal.SaveSnapshot saves the snapshot *position*,
			// not the actual full snapshot data.
			// That happens in sv.saveSnapshot just above.
			// (So don't worry, we're not saving it twice.)
			return wal.SaveSnapshot(walpb.Snapshot{
				Index: rd.Snapshot.Metadata.Index,
				Term:  rd.Snapshot.Metadata.Term,
			})
		})
		err := wal.ReleaseLockTo(rd.Snapshot.Metadata.Index)
		if err != nil {
			panic(err)
		}
		// Only error here is snapshot too old;
		// should be impossible.
		// (And if it happens, it's permanent.)
		err = sv.raftStorage.ApplySnapshot(rd.Snapshot)
		if err != nil {
			panic(err)
		}
		sv.snapIndex = rd.Snapshot.Metadata.Index
		sv.stateMu.Lock()
		sv.confState = rd.Snapshot.Metadata.ConfState
		sv.stateMu.Unlock()
	}
	sv.raftStorage.Append(rd.Entries)
	var lastEntryIndex uint64
	for _, entry := range rd.CommittedEntries {
		sv.applyEntry(entry, writers)
		lastEntryIndex = entry.Index
	}

	// NOTE(kr): we must apply entries before sending messages,
	// because some ConfChangeAddNode entries contain the address
	// needed for subsequent messages.
	sv.send(rd.Messages)
	if lastEntryIndex > sv.snapIndex+snapCount {
		sv.redo(func() error {
			return sv.triggerSnapshot()
		})
	}
	sv.raftNode.Advance()
}

func replyReadIndex(rdIndices map[string]chan uint64, readStates []raft.ReadState) {
	for _, state := range readStates {
		ch, ok := rdIndices[string(state.RequestCtx)]
		if ok {
			ch <- state.Index
			delete(rdIndices, string(state.RequestCtx))
		}
	}
}

// runUpdates runs forever, reading and processing updates from raft
// onto local storage.
func (sv *Service) runUpdates(wal *wal.WAL) {
	defer func() {
		v := recover()
		if err, ok := v.(error); ok {
			sv.errMu.Lock()
			sv.err = err
			sv.errMu.Unlock()
			log.Printf(context.Background(), "raft exiting: %v", err)
			debug.PrintStack()
		} else if v != nil {
			panic(v)
		}

		sv.stateMu.Lock()
		sv.done = true
		sv.stateMu.Unlock()
		sv.stateCond.Broadcast()
	}()
	defer sv.raftNode.Stop()
	defer close(sv.donec)
	defer log.Printf(context.Background(), "ats:got here deferred")

	rdIndices := make(map[string]chan uint64)
	writers := make(map[string]chan bool)
	for {
		select {
		case rd := <-sv.raftNode.Ready():
			replyReadIndex(rdIndices, rd.ReadStates)
			sv.runUpdatesReady(rd, wal, writers)
		case req := <-sv.rctxReq:
			if req.index == nil {
				delete(rdIndices, string(req.rctx))
			} else {
				rdIndices[string(req.rctx)] = req.index
			}
		case req := <-sv.wctxReq:
			if req.satisfied == nil {
				delete(writers, string(req.wctx))
			} else {
				writers[string(req.wctx)] = req.satisfied
			}
		}
	}
}

func runTicks(rn raft.Node) {
	for range time.Tick(tickDur) {
		rn.Tick()
	}
}

// Exec proposes the provided instruction and waits for it to be
// satisfied.
func (sv *Service) Exec(ctx context.Context, instruction []byte) (satisfied bool, err error) {
	if !sv.initialized() {
		return false, ErrUninitialized
	}

	prop := proposal{Wctx: randID(), Instruction: instruction}
	data, err := json.Marshal(prop)
	if err != nil {
		return false, errors.Wrap(err)
	}
	req := wctxReq{wctx: prop.Wctx, satisfied: make(chan bool, 1)}
	select {
	case sv.wctxReq <- req:
	case <-sv.donec:
		return false, errors.New("raft shutdown")
	}
	err = sv.raftNode.Propose(ctx, data)
	if err != nil {
		select {
		case sv.wctxReq <- wctxReq{wctx: prop.Wctx}:
		case <-sv.donec:
		}
		return false, errors.Wrap(err)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute) //TODO(tessr): realistic timeout
	defer cancel()

	select {
	case ok := <-req.satisfied:
		return ok, nil
	case <-ctx.Done():
		return false, ctx.Err()
	case <-sv.donec:
		return false, errors.New("raft shutdown")
	}
}

func (sv *Service) allocNodeID(ctx context.Context) (uint64, error) {
	// lock state via mutex to pull nextID val, then call increment
	var err error
	var satisfied bool
	var nextID, index uint64
	for !satisfied {
		sv.stateMu.Lock()
		nextID, index = sv.state.NextNodeID()
		log.Printf(ctx, "raft: attempting to allocate nodeID %d at version %d", nextID, index)
		sv.stateMu.Unlock()
		b := sv.state.IncrementNextNodeID(nextID, index)
		satisfied, err = sv.Exec(ctx, b)
		if err != nil {
			return 0, err
		}
	}
	return nextID, nil
}

// WaitRead waits for a linearizable read. Upon successful return,
// subsequent reads will observe all writes that happened before the
// call to WaitRead. (There is still no guarantee an intervening Set
// won't have changed the value again, but it is guaranteed not to
// read stale data.)
func (sv *Service) WaitRead(ctx context.Context) error {
	if !sv.initialized() {
		return ErrUninitialized
	}

	for {
		err := sv.waitRead(ctx)
		if isTimeout(err) {
			continue
		}
		return err
	}
}

func (sv *Service) waitRead(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, electionTick*tickDur)
	defer cancel()
	// TODO (ameets)[WIP] possibly refactor, maybe read while holding the lock?
	rctx := randID()
	req := rctxReq{rctx: rctx, index: make(chan uint64, 1)}
	select {
	case sv.rctxReq <- req:
	case <-sv.donec:
		return errors.New("raft shutdown")
	case <-ctx.Done():
		return ctx.Err()
	}
	err := sv.raftNode.ReadIndex(ctx, rctx)
	if err != nil {
		// If we get here, we're going to return this error no matter what.
		// But we need to tell the main loop to delete this read request entry
		// from rdIndices, its read request table, which we do via the following
		// select statement.
		select {
		case sv.rctxReq <- rctxReq{rctx: rctx}:
		case <-sv.donec:
		case <-ctx.Done():
		}
		return err
	}
	// Reads piggyback on writes, so if there's no write traffic, this will never
	// complete. To prevent this, we can send an arbitrary write to Raft if we
	// wait too long, but cancel it if we finish waiting.
	cancelDummy := make(chan struct{})
	go func() {
		select {
		case <-time.After(dummyWriteTimeout):
			satisfied, err := sv.Exec(ctx, sv.state.EmptyWrite())
			if err != nil {
				return // ok to ignore this error, it will retry
			}
			if !satisfied {
				err = errors.New("empty write unsatisfied")
				return
			}
		case <-cancelDummy:
			// We're done waiting.
		}
	}()
	defer close(cancelDummy)

	select {
	case idx := <-req.index:
		ok := sv.wait(idx)
		if !ok {
			return errors.New("raft shutdown")
		}
	case <-sv.donec:
		return errors.New("raft shutdown")
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (sv *Service) wait(index uint64) bool {
	sv.stateMu.Lock()
	defer sv.stateMu.Unlock()
	for !sv.done && sv.state.AppliedIndex() < index {
		sv.stateCond.Wait()
	}
	return sv.state.AppliedIndex() >= index //if false killed b/c of done signal
}

// ServeHTTP responds to raft consensus messages at /raft/x,
// where x is any particular raft internode RPC.
// When sv sends outgoing messages, it acts as an HTTP client
// and sends requests to its peers at /raft/x.
func (sv *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sv.mux.ServeHTTP(w, req)
}

func (sv *Service) serveMsg(w http.ResponseWriter, req *http.Request) {
	if !sv.initialized() {
		errorFormatter.Write(req.Context(), w, ErrUninitialized)
		return
	}

	b, err := ioutil.ReadAll(http.MaxBytesReader(w, req.Body, maxRaftReqSize))
	if err != nil {
		err = errors.Sub(errBadRequest, err)
		errorFormatter.Write(req.Context(), w, err)
		return
	}
	var m raftpb.Message
	err = m.Unmarshal(b)
	if err != nil {
		err = errors.Sub(errBadRequest, err)
		errorFormatter.Write(req.Context(), w, err)
		return
	}
	sv.raftNode.Step(req.Context(), m)
}

func (sv *Service) serveJoin(w http.ResponseWriter, req *http.Request) {
	if !sv.initialized() {
		errorFormatter.Write(req.Context(), w, ErrUninitialized)
		return
	}

	var x struct{ Addr string }
	err := json.NewDecoder(req.Body).Decode(&x)
	if err != nil {
		err = errors.Sub(errBadRequest, err)
		errorFormatter.Write(req.Context(), w, err)
		return
	}

	newID, err := sv.allocNodeID(req.Context())
	if err != nil {
		errorFormatter.Write(req.Context(), w, err)
		return
	}

	// wait before reading so we can perform a linearizable read of
	// the membership list.
	err = sv.WaitRead(req.Context())
	if err != nil {
		errorFormatter.Write(req.Context(), w, err)
		return
	}
	if !sv.state.IsAllowedMember(x.Addr) {
		const detail = "Add this address to the allowed member list before attempting to join the cluster."
		err = errors.WithDetail(ErrAddressNotAllowed, detail)
		errorFormatter.Write(req.Context(), w, err)
		return
	}

	err = sv.raftNode.ProposeConfChange(req.Context(), raftpb.ConfChange{
		ID:      atomic.AddUint64(&sv.confChangeID, 1),
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  newID,
		Context: []byte(x.Addr),
	})
	if err != nil {
		errorFormatter.Write(req.Context(), w, err)
		return
	}

	snap := sv.getSnapshot()
	snapData, err := encodeSnapshot(snap)
	if err != nil {
		errorFormatter.Write(req.Context(), w, err)
		return
	}
	json.NewEncoder(w).Encode(nodeJoin{newID, snapData})
}

// join attempts to join the cluster.
// It requests an existing member to propose a configuration change
// adding the local process as a new member, then retrieves its new ID
// and a snapshot of the cluster state and applies it to sv.
func (sv *Service) join(addr, baseURL string) (id uint64, walobj *wal.WAL, err error) {
	reqURL := strings.TrimRight(baseURL, "/") + "/raft/join"
	b, _ := json.Marshal(struct{ Addr string }{addr})
	resp, err := sv.client.Post(reqURL, contentType, bytes.NewReader(b))
	if err != nil {
		return 0, nil, errors.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("boot server responded with status %d", resp.StatusCode)
		if errResponse, ok := httperror.Parse(resp.Body); ok {
			switch errResponse.ChainCode {
			case "CH162":
				err = errors.WithDetail(ErrAddressNotAllowed, errResponse.Detail)
			case "CH163":
				const detail = "Initialize the boot node before attempting to join its cluster."
				err = errors.WithDetail(ErrPeerUninitialized, detail)
			}
		}
		return 0, nil, errors.Wrap(err, "joining cluster")
	}

	var x nodeJoin
	err = json.NewDecoder(resp.Body).Decode(&x)
	if err != nil {
		return 0, nil, errors.Wrap(err)
	}
	id = x.ID
	var raftSnap raftpb.Snapshot
	err = decodeSnapshot(x.Snap, &raftSnap)
	if err != nil {
		return 0, nil, errors.Wrap(err)
	}

	ctx := context.Background()
	err = sv.raftStorage.ApplySnapshot(raftSnap)
	if err != nil {
		return 0, nil, errors.Wrap(err)
	}

	log.Printkv(ctx, "raftid", id)
	err = writeID(sv.dir, id)
	if err != nil {
		return 0, nil, err
	}

	err = os.Remove(sv.walDir())
	if err != nil {
		return 0, nil, errors.Wrap(err)
	}
	walobj, err = wal.Create(sv.walDir(), nil)
	if err != nil {
		return 0, nil, errors.Wrap(err)
	}

	if !raft.IsEmptySnap(raftSnap) {
		err := sv.saveSnapshot(&raftSnap)
		if err != nil {
			return 0, nil, errors.Wrap(err)
		}
		err = walobj.SaveSnapshot(walpb.Snapshot{
			Index: raftSnap.Metadata.Index,
			Term:  raftSnap.Metadata.Term,
		})
		if err != nil {
			return 0, nil, errors.Wrap(err)
		}
		err = sv.state.RestoreSnapshot(raftSnap.Data, raftSnap.Metadata.Index)
		if err != nil {
			return 0, nil, errors.Wrap(err)
		}
		sv.confState = raftSnap.Metadata.ConfState
		sv.snapIndex = raftSnap.Metadata.Index
	}
	log.Printkv(ctx, "at", "joined", "appliedindex", raftSnap.Metadata.Index)
	return id, walobj, nil
}

func encodeSnapshot(snapshot *raftpb.Snapshot) ([]byte, error) {
	b, err := snapshot.Marshal()
	if err != nil {
		return nil, err
	}

	crc := crc32.Checksum(b, crcTable)
	snap := snappb.Snapshot{Crc: crc, Data: b}
	return snap.Marshal()
}

func decodeSnapshot(data []byte, snapshot *raftpb.Snapshot) error {
	var snapPB snappb.Snapshot
	err := snapPB.Unmarshal(data)
	if err != nil {
		return errors.Wrap(err)
	}
	if crc32.Checksum(snapPB.Data, crcTable) != snapPB.Crc {
		return errors.Wrap(errors.New("bad snapshot crc"))
	}
	err = snapshot.Unmarshal(snapPB.Data)
	return errors.Wrap(err)
}

func (sv *Service) applyEntry(ent raftpb.Entry, writers map[string]chan bool) {
	switch ent.Type {
	case raftpb.EntryConfChange:
		var cc raftpb.ConfChange
		err := cc.Unmarshal(ent.Data)
		if err != nil {
			panic(err)
		}
		sv.stateMu.Lock()
		sv.confState = *sv.raftNode.ApplyConfChange(cc)
		sv.stateMu.Unlock()
		switch cc.Type {
		case raftpb.ConfChangeAddNode, raftpb.ConfChangeUpdateNode:
			sv.stateMu.Lock()
			defer sv.stateMu.Unlock()
			defer sv.stateCond.Broadcast()
			sv.state.SetPeerAddr(cc.NodeID, string(cc.Context))
		case raftpb.ConfChangeRemoveNode:
			if cc.NodeID == sv.id {
				panic(errors.New("removed from cluster"))
			}
			sv.stateMu.Lock()
			defer sv.stateMu.Unlock()
			defer sv.stateCond.Broadcast()
			sv.state.RemovePeerAddr(cc.NodeID)
		}
	case raftpb.EntryNormal:
		//raft will send empty request defaulted to EntryNormal on leader election
		//we need to handle that here
		if ent.Data == nil {
			break
		}
		sv.stateMu.Lock()
		defer sv.stateCond.Broadcast()
		defer sv.stateMu.Unlock()
		var p proposal
		err := json.Unmarshal(ent.Data, &p)
		if err != nil {
			panic(err)
		}
		satisfied := sv.state.Apply(p.Instruction, ent.Index)
		// send 'satisfied' over channel to caller
		if c := writers[string(p.Wctx)]; c != nil {
			c <- satisfied
			delete(writers, string(p.Wctx))
		}
	default:
		panic(fmt.Errorf("unknown entry type: %v", ent))
	}
}

func (sv *Service) send(msgs []raftpb.Message) {
	for _, msg := range msgs {
		data, err := msg.Marshal()
		if err != nil {
			panic(err)
		}
		sv.stateMu.Lock()
		addr := sv.state.GetPeerAddr(msg.To)
		sv.stateMu.Unlock()
		if addr == "" {
			log.Printkv(context.Background(), "no-addr-for-peer", msg.To)
			continue
		}
		sendmsg(addr, data, sv.client)
	}
}

// best effort. if it fails, oh well -- that's why we're using raft.
func sendmsg(addr string, data []byte, client *http.Client) {
	url := "https://" + addr + "/raft/msg"
	resp, err := client.Post(url, contentType, bytes.NewReader(data))
	if err != nil {
		log.Printkv(context.Background(), "warning", err)
		return
	}
	defer resp.Body.Close()
}

// recover loads state from the last full snapshot,
// then replays WAL entries into the raft instance.
// The returned WAL object is nil if no WAL is found.
func (sv *Service) recover() (*wal.WAL, error) {
	err := os.MkdirAll(sv.walDir(), 0777)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var raftSnap raftpb.Snapshot
	snapData, err := ioutil.ReadFile(sv.snapFile())
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err)
	}
	if err == nil {
		err = decodeSnapshot(snapData, &raftSnap)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	if ents, err := ioutil.ReadDir(sv.walDir()); err == nil && len(ents) == 0 {
		return nil, nil
	}

	wal, err := wal.Open(sv.walDir(), walpb.Snapshot{
		Index: raftSnap.Metadata.Index,
		Term:  raftSnap.Metadata.Term,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	_, st, ents, err := wal.ReadAll()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	sv.raftStorage.ApplySnapshot(raftSnap)
	if !raft.IsEmptySnap(raftSnap) {
		err = sv.state.RestoreSnapshot(raftSnap.Data, raftSnap.Metadata.Index)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		sv.confState = raftSnap.Metadata.ConfState
		sv.snapIndex = raftSnap.Metadata.Index
	}

	sv.raftStorage.SetHardState(st)
	sv.raftStorage.Append(ents)
	return wal, nil
}

func (sv *Service) getSnapshot() *raftpb.Snapshot {
	sv.stateMu.Lock()
	defer sv.stateMu.Unlock()
	data, index, err := sv.state.Snapshot()
	if err != nil {
		panic(err)
	}
	snap, err := sv.raftStorage.CreateSnapshot(index, &sv.confState, data)
	if err != nil {
		panic(err)
	}
	return &snap
}

func (sv *Service) triggerSnapshot() error {
	snap := sv.getSnapshot()
	err := sv.saveSnapshot(snap)
	if err != nil {
		return errors.Wrap(err)
	}

	var compactIndex uint64 = 1
	if snap.Metadata.Index > nSnapCatchupEntries {
		compactIndex = snap.Metadata.Index - nSnapCatchupEntries
	}
	err = sv.raftStorage.Compact(compactIndex)
	if err != nil {
		panic(err)
	}
	sv.snapIndex = snap.Metadata.Index
	return nil
}

func (sv *Service) saveSnapshot(snapshot *raftpb.Snapshot) error {
	d, err := encodeSnapshot(snapshot)
	if err != nil {
		panic(err)
	}
	return writeFile(sv.snapFile(), d, 0666)
}

func readID(dir string) (uint64, error) {
	d, err := ioutil.ReadFile(filepath.Join(dir, "id"))
	if err != nil {
		return 0, err
	}
	if len(d) != 12 {
		return 0, errors.New("bad id file size")
	}
	id := binary.BigEndian.Uint64(d)
	if id == 0 {
		return 0, errors.New("invalid id")
	}
	if crc32.Checksum(d[:8], crcTable) != binary.BigEndian.Uint32(d[8:]) {
		return 0, fmt.Errorf("bad CRC in member id %x", d)
	}
	return id, nil
}

func writeID(dir string, id uint64) error {
	b := make([]byte, 12)
	binary.BigEndian.PutUint64(b, id)
	binary.BigEndian.PutUint32(b[8:], crc32.Checksum(b[:8], crcTable))
	name := filepath.Join(dir, "id")
	return errors.Wrap(writeFile(name, b, 0666))
}

func (sv *Service) walDir() string   { return filepath.Join(sv.dir, "wal") }
func (sv *Service) snapFile() string { return filepath.Join(sv.dir, "snap") }

// redo runs f repeatedly until it returns nil, with exponential backoff.
// It reports any errors encountered using sv.Error.
// It must be called nowhere but runUpdates.
func (sv *Service) redo(f func() error) {
	for n := uint(0); ; n++ {
		err := f()
		sv.errMu.Lock()
		sv.err = err
		sv.errMu.Unlock()
		if err == nil {
			break
		}
		time.Sleep(100*time.Millisecond + time.Millisecond<<n)
	}
}

func isTimeout(err error) bool {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		return true
	}

	return err == context.DeadlineExceeded
}
