package main

type changeAddr struct {
	Address        string        `json:"address"`
	AddrComponents []*signerResp `json:"address_components"`
	SigsRequired   int           `json:"signatures_required"`
}

type signerResp struct {
	Entity         string   `json:"entity"`
	Type           string   `json:"type,omitempty"`
	XPubHash       string   `json:"xpub_hash,omitempty"`
	DerivationPath []uint32 `json:"derivation_path,omitempty"`
	PubKey         string   `json:"pubkey"`
}
