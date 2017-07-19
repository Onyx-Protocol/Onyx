package txvm2

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

type record struct {
	commandprogram []byte
	data           item
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

type program struct {
	program []byte
}

type nonce struct {
	program          []byte
	mintime, maxtime int64
	blockchainid     []byte
}

type assetdefinition struct {
	issuanceprogram program
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
