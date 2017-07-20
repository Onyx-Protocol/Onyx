package txvm

import "chain/crypto/ca"

// This file is read by gen.go at "go generate" time and produces
// typegen.go.

type namedtuple interface {
	name() string
	entuple() tuple
	detuple(tuple) bool
}

type txwitness struct {
	version  int64
	runlimit int64
	program  []byte
}

type tx struct {
	version    int64
	runlimit   int64
	effecthash []byte
}

type value struct {
	amount  int64
	assetID []byte
}

type valuecommitment struct {
	vc *ca.ValueCommitment
}

func (v valuecommitment) entuple() tuple {
	return tuple{
		vbytes("valuecommitment"),
		vbytes(v.vc.V().Bytes()),
		vbytes(v.vc.F().Bytes()),
	}
}

func (v *valuecommitment) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "valuecommitment" {
		return false
	}
	v.vc = new(ca.ValueCommitment)
	return v.vc.FromBytes(append(t[1].(vbytes), t[2].(vbytes)...))
}

type assetcommitment struct {
	ac *ca.AssetCommitment
}

func (a assetcommitment) entuple() tuple {
	return tuple{
		vbytes("assetcommitment"),
		vbytes(a.ac.H().Bytes()),
		vbytes(a.ac.C().Bytes()),
	}
}

func (a *assetcommitment) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "assetcommitment" {
		return false
	}
	a.ac = new(ca.AssetCommitment)
	return a.ac.FromBytes(append(t[1].(vbytes), t[2].(vbytes)...))
}

type unprovenvalue struct {
	vc valuecommitment
}

type provenvalue struct {
	vc valuecommitment
	ac assetcommitment
}

type record struct {
	commandprogram []byte
	data           Item
}

type input struct {
	contractid []byte
}

type output struct {
	contractid []byte
}

type read struct {
	contractid []byte
}

type contract struct {
	value   namedtuple // value or provenvalue
	program []byte
	anchor  []byte
}

type program struct {
	program []byte
}

type nonce struct {
	program          []byte
	mintime, maxtime int64
	blockchainid     []byte
}

type anchor struct {
	value []byte
}

type retirement struct {
	vc valuecommitment
}

type assetdefinition struct {
	issuanceprogram []byte
}

type issuancecandidate struct {
	assetID     []byte
	issuanceKey []byte
}

type maxtime struct {
	maxtime int64
}

type mintime struct {
	mintime int64
}

type annotation struct {
	data []byte
}

type legacyoutput struct {
	sourceID []byte
	assetID  []byte
	amount   int64
	index    int64
	program  []byte
	data     []byte
}

type vm1program struct {
	amount      int64
	assetID     []byte
	entryID     []byte
	outputID    []byte
	index       int64
	anchorID    []byte
	entrydata   []byte
	issuanceKey []byte
	program     []byte
}
