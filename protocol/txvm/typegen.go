// Auto-generated from types.go by gen.go

package txvm

var txwitnessType = (*txwitness)(nil)

func (x *txwitness) name() string { return "txwitness" }

func (x *txwitness) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vint64(x.version),
		vint64(x.runlimit),
		vbytes(x.program),
	}
}

func (x *txwitness) detuple(t tuple) bool {
	if len(t) != 4 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.version = int64(t[1].(vint64))
	x.runlimit = int64(t[2].(vint64))
	x.program = []byte(t[3].(vbytes))
	return true
}

func (x *txwitness) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekTxwitness(stacknum int64) *txwitness {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x txwitness
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid txwitness"))
	}
	return &x
}

func (vm *vm) popTxwitness(stacknum int64) *txwitness {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x txwitness
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid txwitness"))
	}
	return &x
}

func (vm *vm) pushTxwitness(stacknum int64, x *txwitness) {
	vm.push(stacknum, x.entuple())
}

var txType = (*tx)(nil)

func (x *tx) name() string { return "tx" }

func (x *tx) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vint64(x.version),
		vint64(x.runlimit),
		vbytes(x.effecthash),
	}
}

func (x *tx) detuple(t tuple) bool {
	if len(t) != 4 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.version = int64(t[1].(vint64))
	x.runlimit = int64(t[2].(vint64))
	x.effecthash = []byte(t[3].(vbytes))
	return true
}

func (x *tx) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekTx(stacknum int64) *tx {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x tx
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid tx"))
	}
	return &x
}

func (vm *vm) popTx(stacknum int64) *tx {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x tx
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid tx"))
	}
	return &x
}

func (vm *vm) pushTx(stacknum int64, x *tx) {
	vm.push(stacknum, x.entuple())
}

var valueType = (*value)(nil)

func (x *value) name() string { return "value" }

func (x *value) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vint64(x.amount),
		vbytes(x.assetID),
	}
}

func (x *value) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.amount = int64(t[1].(vint64))
	x.assetID = []byte(t[2].(vbytes))
	return true
}

func (x *value) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekValue(stacknum int64) *value {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x value
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid value"))
	}
	return &x
}

func (vm *vm) popValue(stacknum int64) *value {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x value
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid value"))
	}
	return &x
}

func (vm *vm) pushValue(stacknum int64, x *value) {
	vm.push(stacknum, x.entuple())
}

var valuecommitmentType = (*valuecommitment)(nil)

func (x *valuecommitment) name() string { return "valuecommitment" }

func (x *valuecommitment) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekValuecommitment(stacknum int64) *valuecommitment {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x valuecommitment
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid valuecommitment"))
	}
	return &x
}

func (vm *vm) popValuecommitment(stacknum int64) *valuecommitment {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x valuecommitment
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid valuecommitment"))
	}
	return &x
}

func (vm *vm) pushValuecommitment(stacknum int64, x *valuecommitment) {
	vm.push(stacknum, x.entuple())
}

var assetcommitmentType = (*assetcommitment)(nil)

func (x *assetcommitment) name() string { return "assetcommitment" }

func (x *assetcommitment) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekAssetcommitment(stacknum int64) *assetcommitment {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x assetcommitment
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid assetcommitment"))
	}
	return &x
}

func (vm *vm) popAssetcommitment(stacknum int64) *assetcommitment {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x assetcommitment
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid assetcommitment"))
	}
	return &x
}

func (vm *vm) pushAssetcommitment(stacknum int64, x *assetcommitment) {
	vm.push(stacknum, x.entuple())
}

var unprovenvalueType = (*unprovenvalue)(nil)

func (x *unprovenvalue) name() string { return "unprovenvalue" }

func (x *unprovenvalue) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		x.vc.entuple(),
	}
}

func (x *unprovenvalue) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	if !x.vc.detuple(t[1].(tuple)) {
		return false
	}
	return true
}

func (x *unprovenvalue) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekUnprovenvalue(stacknum int64) *unprovenvalue {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x unprovenvalue
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid unprovenvalue"))
	}
	return &x
}

func (vm *vm) popUnprovenvalue(stacknum int64) *unprovenvalue {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x unprovenvalue
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid unprovenvalue"))
	}
	return &x
}

func (vm *vm) pushUnprovenvalue(stacknum int64, x *unprovenvalue) {
	vm.push(stacknum, x.entuple())
}

var provenvalueType = (*provenvalue)(nil)

func (x *provenvalue) name() string { return "provenvalue" }

func (x *provenvalue) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		x.vc.entuple(),
		x.ac.entuple(),
	}
}

func (x *provenvalue) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	if !x.vc.detuple(t[1].(tuple)) {
		return false
	}
	if !x.ac.detuple(t[2].(tuple)) {
		return false
	}
	return true
}

func (x *provenvalue) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekProvenvalue(stacknum int64) *provenvalue {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x provenvalue
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid provenvalue"))
	}
	return &x
}

func (vm *vm) popProvenvalue(stacknum int64) *provenvalue {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x provenvalue
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid provenvalue"))
	}
	return &x
}

func (vm *vm) pushProvenvalue(stacknum int64, x *provenvalue) {
	vm.push(stacknum, x.entuple())
}

var recordType = (*record)(nil)

func (x *record) name() string { return "record" }

func (x *record) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.commandprogram),
		x.data,
	}
}

func (x *record) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.commandprogram = []byte(t[1].(vbytes))
	x.data = t[2]
	return true
}

func (x *record) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekRecord(stacknum int64) *record {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x record
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid record"))
	}
	return &x
}

func (vm *vm) popRecord(stacknum int64) *record {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x record
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid record"))
	}
	return &x
}

func (vm *vm) pushRecord(stacknum int64, x *record) {
	vm.push(stacknum, x.entuple())
}

var inputType = (*input)(nil)

func (x *input) name() string { return "input" }

func (x *input) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.contractid),
	}
}

func (x *input) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.contractid = []byte(t[1].(vbytes))
	return true
}

func (x *input) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekInput(stacknum int64) *input {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x input
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid input"))
	}
	return &x
}

func (vm *vm) popInput(stacknum int64) *input {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x input
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid input"))
	}
	return &x
}

func (vm *vm) pushInput(stacknum int64, x *input) {
	vm.push(stacknum, x.entuple())
}

var outputType = (*output)(nil)

func (x *output) name() string { return "output" }

func (x *output) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.contractid),
	}
}

func (x *output) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.contractid = []byte(t[1].(vbytes))
	return true
}

func (x *output) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekOutput(stacknum int64) *output {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x output
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid output"))
	}
	return &x
}

func (vm *vm) popOutput(stacknum int64) *output {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x output
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid output"))
	}
	return &x
}

func (vm *vm) pushOutput(stacknum int64, x *output) {
	vm.push(stacknum, x.entuple())
}

var readType = (*read)(nil)

func (x *read) name() string { return "read" }

func (x *read) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.contractid),
	}
}

func (x *read) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.contractid = []byte(t[1].(vbytes))
	return true
}

func (x *read) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekRead(stacknum int64) *read {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x read
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid read"))
	}
	return &x
}

func (vm *vm) popRead(stacknum int64) *read {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x read
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid read"))
	}
	return &x
}

func (vm *vm) pushRead(stacknum int64, x *read) {
	vm.push(stacknum, x.entuple())
}

var contractType = (*contract)(nil)

func (x *contract) name() string { return "contract" }

func (x *contract) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		x.value.entuple(),
		vbytes(x.program),
		vbytes(x.anchor),
	}
}

func (x *contract) detuple(t tuple) bool {
	if len(t) != 4 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	if !x.value.detuple(t[1].(tuple)) {
		return false
	}
	x.program = []byte(t[2].(vbytes))
	x.anchor = []byte(t[3].(vbytes))
	return true
}

func (x *contract) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekContract(stacknum int64) *contract {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x contract
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid contract"))
	}
	return &x
}

func (vm *vm) popContract(stacknum int64) *contract {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x contract
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid contract"))
	}
	return &x
}

func (vm *vm) pushContract(stacknum int64, x *contract) {
	vm.push(stacknum, x.entuple())
}

var programType = (*program)(nil)

func (x *program) name() string { return "program" }

func (x *program) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.program),
	}
}

func (x *program) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.program = []byte(t[1].(vbytes))
	return true
}

func (x *program) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekProgram(stacknum int64) *program {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x program
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid program"))
	}
	return &x
}

func (vm *vm) popProgram(stacknum int64) *program {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x program
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid program"))
	}
	return &x
}

func (vm *vm) pushProgram(stacknum int64, x *program) {
	vm.push(stacknum, x.entuple())
}

var nonceType = (*nonce)(nil)

func (x *nonce) name() string { return "nonce" }

func (x *nonce) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.program),
		vint64(x.mintime),
		vint64(x.maxtime),
		vbytes(x.blockchainid),
	}
}

func (x *nonce) detuple(t tuple) bool {
	if len(t) != 5 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.program = []byte(t[1].(vbytes))
	x.mintime = int64(t[2].(vint64))
	x.maxtime = int64(t[3].(vint64))
	x.blockchainid = []byte(t[4].(vbytes))
	return true
}

func (x *nonce) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekNonce(stacknum int64) *nonce {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x nonce
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid nonce"))
	}
	return &x
}

func (vm *vm) popNonce(stacknum int64) *nonce {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x nonce
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid nonce"))
	}
	return &x
}

func (vm *vm) pushNonce(stacknum int64, x *nonce) {
	vm.push(stacknum, x.entuple())
}

var anchorType = (*anchor)(nil)

func (x *anchor) name() string { return "anchor" }

func (x *anchor) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.value),
	}
}

func (x *anchor) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.value = []byte(t[1].(vbytes))
	return true
}

func (x *anchor) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekAnchor(stacknum int64) *anchor {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x anchor
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid anchor"))
	}
	return &x
}

func (vm *vm) popAnchor(stacknum int64) *anchor {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x anchor
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid anchor"))
	}
	return &x
}

func (vm *vm) pushAnchor(stacknum int64, x *anchor) {
	vm.push(stacknum, x.entuple())
}

var retirementType = (*retirement)(nil)

func (x *retirement) name() string { return "retirement" }

func (x *retirement) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		x.vc.entuple(),
	}
}

func (x *retirement) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	if !x.vc.detuple(t[1].(tuple)) {
		return false
	}
	return true
}

func (x *retirement) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekRetirement(stacknum int64) *retirement {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x retirement
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid retirement"))
	}
	return &x
}

func (vm *vm) popRetirement(stacknum int64) *retirement {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x retirement
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid retirement"))
	}
	return &x
}

func (vm *vm) pushRetirement(stacknum int64, x *retirement) {
	vm.push(stacknum, x.entuple())
}

var assetdefinitionType = (*assetdefinition)(nil)

func (x *assetdefinition) name() string { return "assetdefinition" }

func (x *assetdefinition) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.issuanceprogram),
	}
}

func (x *assetdefinition) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.issuanceprogram = []byte(t[1].(vbytes))
	return true
}

func (x *assetdefinition) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekAssetdefinition(stacknum int64) *assetdefinition {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x assetdefinition
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid assetdefinition"))
	}
	return &x
}

func (vm *vm) popAssetdefinition(stacknum int64) *assetdefinition {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x assetdefinition
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid assetdefinition"))
	}
	return &x
}

func (vm *vm) pushAssetdefinition(stacknum int64, x *assetdefinition) {
	vm.push(stacknum, x.entuple())
}

var issuancecandidateType = (*issuancecandidate)(nil)

func (x *issuancecandidate) name() string { return "issuancecandidate" }

func (x *issuancecandidate) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.assetID),
		vbytes(x.issuanceKey),
	}
}

func (x *issuancecandidate) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.assetID = []byte(t[1].(vbytes))
	x.issuanceKey = []byte(t[2].(vbytes))
	return true
}

func (x *issuancecandidate) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekIssuancecandidate(stacknum int64) *issuancecandidate {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x issuancecandidate
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid issuancecandidate"))
	}
	return &x
}

func (vm *vm) popIssuancecandidate(stacknum int64) *issuancecandidate {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x issuancecandidate
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid issuancecandidate"))
	}
	return &x
}

func (vm *vm) pushIssuancecandidate(stacknum int64, x *issuancecandidate) {
	vm.push(stacknum, x.entuple())
}

var maxtimeType = (*maxtime)(nil)

func (x *maxtime) name() string { return "maxtime" }

func (x *maxtime) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vint64(x.maxtime),
	}
}

func (x *maxtime) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.maxtime = int64(t[1].(vint64))
	return true
}

func (x *maxtime) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekMaxtime(stacknum int64) *maxtime {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x maxtime
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid maxtime"))
	}
	return &x
}

func (vm *vm) popMaxtime(stacknum int64) *maxtime {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x maxtime
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid maxtime"))
	}
	return &x
}

func (vm *vm) pushMaxtime(stacknum int64, x *maxtime) {
	vm.push(stacknum, x.entuple())
}

var mintimeType = (*mintime)(nil)

func (x *mintime) name() string { return "mintime" }

func (x *mintime) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vint64(x.mintime),
	}
}

func (x *mintime) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.mintime = int64(t[1].(vint64))
	return true
}

func (x *mintime) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekMintime(stacknum int64) *mintime {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x mintime
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid mintime"))
	}
	return &x
}

func (vm *vm) popMintime(stacknum int64) *mintime {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x mintime
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid mintime"))
	}
	return &x
}

func (vm *vm) pushMintime(stacknum int64, x *mintime) {
	vm.push(stacknum, x.entuple())
}

var annotationType = (*annotation)(nil)

func (x *annotation) name() string { return "annotation" }

func (x *annotation) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.data),
	}
}

func (x *annotation) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.data = []byte(t[1].(vbytes))
	return true
}

func (x *annotation) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekAnnotation(stacknum int64) *annotation {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x annotation
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid annotation"))
	}
	return &x
}

func (vm *vm) popAnnotation(stacknum int64) *annotation {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x annotation
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid annotation"))
	}
	return &x
}

func (vm *vm) pushAnnotation(stacknum int64, x *annotation) {
	vm.push(stacknum, x.entuple())
}

var legacyoutputType = (*legacyoutput)(nil)

func (x *legacyoutput) name() string { return "legacyoutput" }

func (x *legacyoutput) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vbytes(x.sourceID),
		vbytes(x.assetID),
		vint64(x.amount),
		vint64(x.index),
		vbytes(x.program),
		vbytes(x.data),
	}
}

func (x *legacyoutput) detuple(t tuple) bool {
	if len(t) != 7 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.sourceID = []byte(t[1].(vbytes))
	x.assetID = []byte(t[2].(vbytes))
	x.amount = int64(t[3].(vint64))
	x.index = int64(t[4].(vint64))
	x.program = []byte(t[5].(vbytes))
	x.data = []byte(t[6].(vbytes))
	return true
}

func (x *legacyoutput) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekLegacyoutput(stacknum int64) *legacyoutput {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x legacyoutput
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid legacyoutput"))
	}
	return &x
}

func (vm *vm) popLegacyoutput(stacknum int64) *legacyoutput {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x legacyoutput
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid legacyoutput"))
	}
	return &x
}

func (vm *vm) pushLegacyoutput(stacknum int64, x *legacyoutput) {
	vm.push(stacknum, x.entuple())
}

var vm1programType = (*vm1program)(nil)

func (x *vm1program) name() string { return "vm1program" }

func (x *vm1program) entuple() tuple {
	return tuple{
		vbytes(x.name()),
		vint64(x.amount),
		vbytes(x.assetID),
		vbytes(x.entryID),
		vbytes(x.outputID),
		vint64(x.index),
		vbytes(x.anchorID),
		vbytes(x.entrydata),
		vbytes(x.issuanceKey),
		vbytes(x.program),
	}
}

func (x *vm1program) detuple(t tuple) bool {
	if len(t) != 10 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != x.name() {
		return false
	}
	x.amount = int64(t[1].(vint64))
	x.assetID = []byte(t[2].(vbytes))
	x.entryID = []byte(t[3].(vbytes))
	x.outputID = []byte(t[4].(vbytes))
	x.index = int64(t[5].(vint64))
	x.anchorID = []byte(t[6].(vbytes))
	x.entrydata = []byte(t[7].(vbytes))
	x.issuanceKey = []byte(t[8].(vbytes))
	x.program = []byte(t[9].(vbytes))
	return true
}

func (x *vm1program) id() []byte { return getID(x.entuple()) }

func (vm *vm) peekVm1program(stacknum int64) *vm1program {
	v := vm.peek(stacknum)
	t := v.(tuple)
	var x vm1program
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid vm1program"))
	}
	return &x
}

func (vm *vm) popVm1program(stacknum int64) *vm1program {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x vm1program
	if !x.detuple(t) {
		panic(vm.err("tuple is not a valid vm1program"))
	}
	return &x
}

func (vm *vm) pushVm1program(stacknum int64, x *vm1program) {
	vm.push(stacknum, x.entuple())
}
