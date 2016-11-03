// Package raft provides a simple key-value store coordinated
// across a raft cluster.
package raft

// tktk need to vendor raft package before landing this

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
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

	"chain/database/raft/state"
	"chain/errors"
	"chain/log"
)

var ErrUnsatisfied = errors.New("precondition not satisfied")

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

var crcTable = crc32.MakeTable(crc32.Castagnoli)

// Service holds the key-value data and performs raft coordination.
type Service struct {
	// config
	dir     string
	id      uint64
	mux     *http.ServeMux
	rctxReq chan rctxReq
	wctxReq chan wctxReq
	donec   chan struct{}

	errMu sync.Mutex
	err   error

	confChangeID uint64 // atomic access only

	// The storage object is purely for internal use
	// by the raft.Node to maintain consistent persistent state.
	// All client code accesses the cluster state via our Service
	// object, which keeps a local, in-memory copy of the
	// complete current state.
	raftNode    raft.Node
	raftStorage *raft.MemoryStorage

	// The actual replicated data set.
	stateMu   sync.Mutex
	stateCond sync.Cond
	state     *state.State
	done      bool

	// Current log position.
	// access only from runUpdates goroutine
	snapIndex uint64
	confState raftpb.ConfState
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

// Getter gets a value from a key-value store.
type Getter interface {
	Get(key string) (value []byte)
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
// for this raft node. If it doesn't exist, Start will create it.
// It has three entries:
//   id    file containing the node's member id (never changes)
//   snap  file containing the last complete state snapshot
//   wal   dir containing the write-ahead log
//
// Param bootURL gives the location of an existing cluster
// for the local process to join.
// It can be either the concrete address of any
// single cluster member or it can point to a load balancer
// for the whole cluster, if one exists.
// An empty bootURL means to start a fresh empty cluster.
// It is ignored when recovering from existing state in dir.
func Start(laddr, dir, bootURL string) (*Service, error) {
	ctx := context.Background()

	sv := &Service{
		dir:         dir,
		mux:         http.NewServeMux(),
		raftStorage: raft.NewMemoryStorage(),
		state:       state.New(),
		donec:       make(chan struct{}),
		rctxReq:     make(chan rctxReq),
		wctxReq:     make(chan wctxReq),
	}
	sv.stateCond.L = &sv.stateMu

	// TODO(kr): grpc
	sv.mux.HandleFunc("/raft/join", sv.serveJoin)
	sv.mux.HandleFunc("/raft/msg", sv.serveMsg)

	walobj, err := sv.recover()
	if err != nil {
		return nil, err
	}

	if walobj != nil {
		sv.id, err = readID(sv.dir)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	} else if bootURL != "" {
		err = sv.join(laddr, bootURL) // sets sv.id and state
		if err != nil {
			return nil, err
		}
	} else {
		// brand new cluster!
		sv.id = 1
	}

	c := &raft.Config{
		ID:              sv.id,
		ElectionTick:    electionTick,
		HeartbeatTick:   heartbeatTick,
		Storage:         sv.raftStorage,
		Applied:         sv.state.AppliedIndex(),
		MaxSizePerMsg:   4096,
		MaxInflightMsgs: 256,
	}

	if walobj != nil {
		sv.raftNode = raft.RestartNode(c)
		// trigger an election in case raft thinks there's no leader
		err = sv.raftNode.Campaign(context.Background())
		if err != nil {
			// oh well, we will campaign anyway after ElectionTick
			log.Error(ctx, err)
		}
	} else if bootURL != "" {
		log.Write(ctx, "raftid", c.ID)
		err = writeID(sv.dir, c.ID)
		if err != nil {
			return nil, err
		}
		walobj, err = wal.Create(sv.walDir(), nil)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		sv.raftNode = raft.RestartNode(c)
	} else {
		log.Write(ctx, "raftid", c.ID)
		err = writeID(sv.dir, c.ID)
		if err != nil {
			return nil, err
		}
		walobj, err = wal.Create(sv.walDir(), nil)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		sv.raftNode = raft.StartNode(c, []raft.Peer{{ID: 1, Context: []byte(laddr)}})
	}

	go sv.runUpdates(walobj)
	go runTicks(sv.raftNode)
	return sv, nil
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
		sv.confState = rd.Snapshot.Metadata.ConfState
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
			log.Messagef(context.Background(), "raft exiting: %v", err)
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
	defer log.Messagef(context.Background(), "ats:got here deferred")

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

//
func (sv *Service) exec(ctx context.Context, instruction []byte) error {
	prop := proposal{Wctx: randID(), Instruction: instruction}
	data, err := json.Marshal(prop)
	if err != nil {
		return errors.Wrap(err)
	}
	req := wctxReq{wctx: prop.Wctx, satisfied: make(chan bool, 1)} //buffered channel
	select {
	case sv.wctxReq <- req:
	case <-sv.donec:
		return errors.New("raft shutdown")
	}
	err = sv.raftNode.Propose(ctx, data)
	if err != nil {
		select {
		case sv.wctxReq <- wctxReq{wctx: prop.Wctx}:
		case <-sv.donec:
		}
		return errors.Wrap(err)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute) //TODO realistic timeout
	defer cancel()

	select {
	case ok := <-req.satisfied:
		if !ok {
			return ErrUnsatisfied
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-sv.donec:
		return errors.New("raft shutdown")
	}
}

// Set sets a value in the key-value storage.
// If successful, it returns after the value is committed to
// the raft log.
// TODO (ameets): possibly RawNode in future to know whether Proposal worked or not
func (sv *Service) Set(ctx context.Context, key string, val []byte) error {
	// encode w/ json for now: b with a wctx rand id
	b := state.Set(key, val)
	return sv.exec(ctx, b)
}

// Insert inserts a value into key-value storage.
// It will fail if there's already a value stored at the given key.
// If successful, it returns after the value is committed to the raft log.
func (sv *Service) Insert(ctx context.Context, key string, val []byte) error {
	b := state.Insert(key, val)
	return sv.exec(ctx, b)
}

// Delete deletes a value in the key-value storage.
// if successful, it returns after the value is deleted from the raft log.
// TODO (ameets): is RawNode possible/applicable?
func (sv *Service) Delete(ctx context.Context, key string) error {
	b := state.Delete(key)
	return sv.exec(ctx, b)
}

func (sv *Service) allocNodeID(ctx context.Context) (uint64, error) {
	// encode w/ json for now: b with a wctx rand id
	// lock state via mutex to pull nextID val, then call increment
	err := ErrUnsatisfied
	var nextID, index uint64
	for err == ErrUnsatisfied {
		sv.stateMu.Lock()
		nextID, index = sv.state.NextNodeID()
		log.Messagef(ctx, "raft: attempting to allocate nodeID %d at version %d", nextID, index)
		sv.stateMu.Unlock()
		b := state.IncrementNextNodeID(nextID, index)
		err = sv.exec(ctx, b)
	}
	return nextID, err //caller should check for error b/c value of nextID is untrustworthy in that case
}

// Get gets a value from the key-value store.
// It is linearizable; that is, if a
// Set happens before a Get,
// the Get will observe the effects of the Set.
// (There is still no guarantee an intervening
// Set won't have changed the value again, but it is
// guaranteed not to read stale data.)
// This can be slow; for faster but possibly stale reads, see Stale.
func (sv *Service) Get(ctx context.Context, key string) ([]byte, error) {
	for {
		resp, err := sv.get(ctx, key)
		if isTimeout(err) {
			continue
		}
		return resp, err
	}
}

func (sv *Service) get(ctx context.Context, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, electionTick*tickDur)
	defer cancel()
	// TODO (ameets)[WIP] possibly refactor, maybe read while holding the lock?
	rctx := randID()
	req := rctxReq{rctx: rctx, index: make(chan uint64, 1)}
	select {
	case sv.rctxReq <- req:
	case <-sv.donec:
		return nil, errors.New("raft shutdown")
	case <-ctx.Done():
		return nil, ctx.Err()
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
		return nil, err
	}
	// Reads piggyback on writes, so if there's no write traffic, this will never
	// complete. To prevent this, we can send an arbitrary write to Raft if we
	// wait too long, but cancel it if we finish waiting.
	cancelDummy := make(chan struct{})
	go func() {
		select {
		case <-time.After(dummyWriteTimeout):
			err := sv.Set(ctx, "/dummyWrite", []byte(""))
			if err != nil {
				// TODO(tessr): should probably return this error
				log.Messagef(ctx, "dummy write failed with %s", err)
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
			return nil, errors.New("raft shutdown")
		}
	case <-sv.donec:
		return nil, errors.New("raft shutdown")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return sv.Stale().Get(key), nil
}

func (sv *Service) wait(index uint64) bool {
	sv.stateMu.Lock()
	defer sv.stateMu.Unlock()
	for !sv.done && sv.state.AppliedIndex() < index {
		sv.stateCond.Wait()
	}
	return sv.state.AppliedIndex() >= index //if false killed b/c of done signal
}

// Stale returns an object that reads
// directly from local memory, returning (possibly) stale data.
// Calls to sv.Get are linearizable,
// which requires them to go through the raft protocol.
// The stale getter skips this, so it is much faster,
// but it can only be used in situations that don't require
// linearizability.
func (sv *Service) Stale() Getter {
	return (*staleGetter)(sv)
}

// ServeHTTP responds to raft consensus messages at /raft/x,
// where x is any particular raft internode RPC.
// When sv sends outgoing messages, it acts as an HTTP client
// and sends requests to its peers at /raft/x.
func (sv *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sv.mux.ServeHTTP(w, req)
}

func (sv *Service) serveMsg(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(http.MaxBytesReader(w, req.Body, maxRaftReqSize))
	if err != nil {
		http.Error(w, "cannot read req: "+err.Error(), 400)
		return
	}
	var m raftpb.Message
	err = m.Unmarshal(b)
	if err != nil {
		http.Error(w, "cannot unmarshal: "+err.Error(), 400)
		return
	}
	sv.raftNode.Step(req.Context(), m)
}

func (sv *Service) serveJoin(w http.ResponseWriter, req *http.Request) {
	var x struct{ Addr string }
	err := json.NewDecoder(req.Body).Decode(&x)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	log.Write(req.Context(), "at", "join", "addr", x.Addr)

	newID, err := sv.allocNodeID(req.Context())
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	log.Write(req.Context(), "at", "join-id", "addr", x.Addr, "id", newID)

	err = sv.raftNode.ProposeConfChange(req.Context(), raftpb.ConfChange{
		ID:      atomic.AddUint64(&sv.confChangeID, 1),
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  newID,
		Context: []byte(x.Addr),
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	log.Write(req.Context(), "at", "join-done", "addr", x.Addr, "id", newID)

	snap := sv.getSnapshot()
	snapData, err := encodeSnapshot(snap)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(struct {
		ID   uint64
		Snap []byte
	}{newID, snapData})
}

// join attempts to join the cluster.
// It requests an existing member to propose a configuration change
// adding the local process as a new member, then retrieves its new ID
// and a snapshot of the cluster state and applies it to sv.
// It also sets sv.id.
func (sv *Service) join(addr, baseURL string) error {
	reqURL := strings.TrimRight(baseURL, "/") + "/raft/join"
	b, _ := json.Marshal(struct{ Addr string }{addr})
	resp, err := http.Post(reqURL, contentType, bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err)
	}
	defer resp.Body.Close()

	var x struct {
		ID   uint64
		Snap []byte
	}
	err = json.NewDecoder(resp.Body).Decode(&x)
	if err != nil {
		return errors.Wrap(err)
	}
	sv.id = x.ID
	var raftSnap raftpb.Snapshot
	err = decodeSnapshot(x.Snap, &raftSnap)
	if err != nil {
		return errors.Wrap(err)
	}

	ctx := context.Background()
	sv.raftStorage.ApplySnapshot(raftSnap)
	if raft.IsEmptySnap(raftSnap) {
		log.Write(ctx, "at", "joined", "appliedindex", 0, "msg", "(empty snap)")
	} else {
		err = sv.state.RestoreSnapshot(raftSnap.Data, raftSnap.Metadata.Index)
		if err != nil {
			return errors.Wrap(err)
		}
		sv.confState = raftSnap.Metadata.ConfState
		sv.snapIndex = raftSnap.Metadata.Index
		log.Write(ctx, "at", "joined", "appliedindex", raftSnap.Metadata.Index)
	}

	return nil
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

// TODO(kr): under what circumstances should we do this?
func (sv *Service) evict(nodeID uint64) {
	//cc := raftpb.ConfChange{
	//	Type:   raftpb.ConfChangeRemoveNode,
	//	NodeID: nodeID,
	//}
}

func (sv *Service) applyEntry(ent raftpb.Entry, writers map[string]chan bool) {
	log.Write(context.Background(), "index", ent.Index, "ent", ent)

	switch ent.Type {
	case raftpb.EntryConfChange:
		var cc raftpb.ConfChange
		err := cc.Unmarshal(ent.Data)
		if err != nil {
			panic(err)
		}
		sv.raftNode.ApplyConfChange(cc)
		switch cc.Type {
		case raftpb.ConfChangeAddNode, raftpb.ConfChangeUpdateNode:
			log.Write(context.Background(),
				"what", cc.Type,
				"id", cc.NodeID,
				"addr", string(cc.Context),
			)
			sv.stateMu.Lock()
			defer sv.stateMu.Unlock()
			defer sv.stateCond.Broadcast()
			sv.state.SetPeerAddr(cc.NodeID, string(cc.Context))
		case raftpb.ConfChangeRemoveNode:
			if cc.NodeID == sv.id {
				log.Write(context.Background(), "nodeID", cc.NodeID, "msg", "removed from cluster")
				panic(errors.New("removed from cluster"))
			}
			sv.stateMu.Lock()
			defer sv.stateMu.Unlock()
			defer sv.stateCond.Broadcast()
			sv.state.RemovePeerAddr(cc.NodeID)
		}
	case raftpb.EntryNormal:
		log.Write(context.Background(), "EntryNormal", ent)
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
			log.Messagef(context.Background(), "ent.data: %q", ent.Data)
			panic(err)
		}
		satisfied, err := sv.state.Apply(p.Instruction, ent.Index)
		if err != nil {
			panic(err)
		}
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
			log.Write(context.Background(), "no-addr-for-peer", msg.To)
			continue
		}
		sendmsg(addr, data)
	}
}

// best effort. if it fails, oh well -- that's why we're using raft.
func sendmsg(addr string, data []byte) {
	resp, err := http.Post("http://"+addr+"/raft/msg", contentType, bytes.NewReader(data))
	if err != nil {
		log.Error(context.Background(), err)
		return
	}
	defer resp.Body.Close()
}

// recover loads state from the last full snapshot,
// then replays WAL entries into the raft instance.
// The returned WAL object is nil if no WAL is found.
func (sv *Service) recover() (*wal.WAL, error) {
	log.Write(context.Background(), "waldir", sv.walDir())
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
	if raft.IsEmptySnap(raftSnap) {
	} else {
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
	data, index, err := sv.state.Snapshot()
	sv.stateMu.Unlock()
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

type staleGetter Service

func (g *staleGetter) Get(key string) []byte {
	g.stateMu.Lock()
	defer g.stateMu.Unlock()
	return g.state.Get(key)
}

func isTimeout(err error) bool {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		return true
	}

	return err == context.DeadlineExceeded
}
