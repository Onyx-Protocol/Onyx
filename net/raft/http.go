package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/coreos/etcd/raft/raftpb"

	"chain/errors"
	"chain/log"
	"chain/net/http/httperror"
)

const (
	contentType = "application/octet-stream"
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

// nodeJoinRequest is the request format of the /raft/join endpoint
// used when a new node joins an existing cluster.
type nodeJoinRequest struct {
	Addr string
}

// nodeJoinResponse is the response format of the /raft/join endpoint
// used when a new node joins an existing cluster.
type nodeJoinResponse struct {
	ID   uint64
	Snap []byte
}

// ServeHTTP responds to raft consensus messages at /raft/x,
// where x is any raft internode RPC.
// When sv sends outgoing messages, it acts as an HTTP client
// and sends requests to its peers at /raft/x.
func (sv *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sv.mux.ServeHTTP(w, req)
}

// serveMsg is registered as a handler for the /raft/msg rpc.
// It accepts a raft message from another node and passes it to
// the raft state machine.
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

	// If message is from node not in cluster, tell node to remove itself
	if sv.state.Peers()[m.From] == "" {
		cc := raftpb.ConfChange{
			ID:     atomic.AddUint64(&sv.confChangeID, 1),
			Type:   raftpb.ConfChangeRemoveNode,
			NodeID: m.From,
		}
		data, err := cc.Marshal()
		if err != nil {
			panic(err)
		}
		sv.send([]raftpb.Message{{
			Type:    raftpb.MsgProp,
			Entries: []raftpb.Entry{{Type: raftpb.EntryConfChange, Data: data}},
		}})
	}
	sv.raftNode.Step(req.Context(), m)
}

// serveJoin is registered as a handler for the /raft/join rpc.
// If the provided node's address is allowed, it adds the node
// to the cluster membership and returns a snapshot through which
// the new node can catch up to the current state.
func (sv *Service) serveJoin(w http.ResponseWriter, req *http.Request) {
	if !sv.initialized() {
		errorFormatter.Write(req.Context(), w, ErrUninitialized)
		return
	}

	var requestBody nodeJoinRequest
	err := json.NewDecoder(req.Body).Decode(&requestBody)
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
	if !sv.state.IsAllowedMember(requestBody.Addr) {
		const detail = "Add this address to the allowed member list before attempting to join the cluster."
		err = errors.WithDetail(ErrAddressNotAllowed, detail)
		errorFormatter.Write(req.Context(), w, err)
		return
	}

	err = sv.raftNode.ProposeConfChange(req.Context(), raftpb.ConfChange{
		ID:      atomic.AddUint64(&sv.confChangeID, 1),
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  newID,
		Context: []byte(requestBody.Addr),
	})
	if err != nil {
		errorFormatter.Write(req.Context(), w, err)
		return
	}

	// Wait for the conf change to be committed. This ensures that the we don't
	// misleadingly tell a node that they successfully joined when the change
	// never commits. It also ensures that the provided snapshot includes the
	// new node.
	// https://github.com/chain/chain/issues/1330
	sv.waitForNode(newID)

	snap := sv.getSnapshot()
	snapData, err := encodeSnapshot(snap)
	if err != nil {
		errorFormatter.Write(req.Context(), w, err)
		return
	}
	json.NewEncoder(w).Encode(nodeJoinResponse{
		ID:   newID,
		Snap: snapData,
	})
}

// best effort. if it fails, oh well -- that's why we're using raft.
func sendmsg(addr string, data []byte, client *http.Client) {
	// TODO(jackson): Parse the error response and try to detect
	// eviction.
	url := "https://" + addr + "/raft/msg"
	resp, err := client.Post(url, contentType, bytes.NewReader(data))
	if err != nil {
		log.Printkv(context.Background(), "warning", err)
		return
	}
	defer resp.Body.Close()
}

func requestJoin(addr, baseURL string, client *http.Client) (*nodeJoinResponse, error) {
	reqURL := strings.TrimRight(baseURL, "/") + "/raft/join"
	b, err := json.Marshal(nodeJoinRequest{Addr: addr})
	if err != nil {
		panic(err)
	}
	resp, err := client.Post(reqURL, contentType, bytes.NewReader(b))
	if err != nil {
		return nil, errors.Wrap(err)
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
		return nil, errors.Wrap(err, "joining cluster")
	}

	parsedResponse := new(nodeJoinResponse)
	err = json.NewDecoder(resp.Body).Decode(parsedResponse)
	return parsedResponse, errors.Wrap(err)
}
