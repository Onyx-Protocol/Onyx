package utxodb

import (
	"sync"

	"chain/fedchain/bc"
)

var (
	// commonID interns common ID strings.
	commonID   = make(map[string]string)
	commonHash = make(map[bc.AssetID]bc.AssetID)
	commonMu   sync.Mutex
)

// caller must hold commonMu.
func intern(s string) string {
	if s := commonID[s]; s != "" {
		return s
	}
	commonID[s] = s
	return s
}

// caller must hold commonMu.
func internHash(a bc.AssetID) bc.AssetID {
	if a, ok := commonHash[a]; ok {
		return a
	}
	commonHash[a] = a
	return a
}

// internIDs interns fields AccountID and AssetID
// for each UTXO in us.
func internIDs(us []*UTXO) {
	commonMu.Lock()
	defer commonMu.Unlock()
	for _, u := range us {
		u.AssetID = internHash(u.AssetID)
		u.AccountID = intern(u.AccountID)
	}
}
