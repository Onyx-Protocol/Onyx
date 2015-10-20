package utxodb

import "sync"

var (
	// commonID interns common ID strings.
	commonID = make(map[string]string)
	commonMu sync.Mutex
)

// caller must hold commonMu.
func intern(s string) string {
	if s := commonID[s]; s != "" {
		return s
	}
	commonID[s] = s
	return s
}

// internIDs interns fields AccountID and AssetID
// for each UTXO in us.
func internIDs(us []*UTXO) {
	commonMu.Lock()
	defer commonMu.Unlock()
	for _, u := range us {
		u.AssetID = intern(u.AssetID)
		u.AccountID = intern(u.AccountID)
	}
}
