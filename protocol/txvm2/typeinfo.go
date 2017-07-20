// Auto-generated from types.go by gen.go

package txvm2

var txwitnessType = (*txwitness)(nil)

func (x txwitness) entuple() tuple {
	return tuple{
		vbytes("txwitness"),
		vint64(x.version),
		vint64(x.runlimit),
		vbytes(x.program),
	}
}

func (x *txwitness) detuple(t tuple) bool {
	if len(t) != 4 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "txwitness" {
		return false
	}
	x.version = int64(t[1].(vint64))
	x.runlimit = int64(t[2].(vint64))
	x.program = []byte(t[3].(vbytes))
	return true
}

func (vm *vm) popTxwitness(stacknum int) txwitness {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x txwitness
	if !x.detuple(t) {
		panic("tuple is not a valid txwitness")
	}
	return x
}

func (vm *vm) pushTxwitness(stacknum int, x txwitness) {
	vm.push(stacknum, x.entuple())
}

var txType = (*tx)(nil)

func (x tx) entuple() tuple {
	return tuple{
		vbytes("tx"),
		vint64(x.version),
		vint64(x.runlimit),
		vbytes(x.effecthash),
	}
}

func (x *tx) detuple(t tuple) bool {
	if len(t) != 4 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "tx" {
		return false
	}
	x.version = int64(t[1].(vint64))
	x.runlimit = int64(t[2].(vint64))
	x.effecthash = []byte(t[3].(vbytes))
	return true
}

func (vm *vm) popTx(stacknum int) tx {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x tx
	if !x.detuple(t) {
		panic("tuple is not a valid tx")
	}
	return x
}

func (vm *vm) pushTx(stacknum int, x tx) {
	vm.push(stacknum, x.entuple())
}

var valueType = (*value)(nil)

func (x value) entuple() tuple {
	return tuple{
		vbytes("value"),
		vint64(x.amount),
		vbytes(x.assetID),
	}
}

func (x *value) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "value" {
		return false
	}
	x.amount = int64(t[1].(vint64))
	x.assetID = []byte(t[2].(vbytes))
	return true
}

func (vm *vm) popValue(stacknum int) value {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x value
	if !x.detuple(t) {
		panic("tuple is not a valid value")
	}
	return x
}

func (vm *vm) pushValue(stacknum int, x value) {
	vm.push(stacknum, x.entuple())
}

var valuecommitmentType = (*valuecommitment)(nil)

func (vm *vm) popValuecommitment(stacknum int) valuecommitment {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x valuecommitment
	if !x.detuple(t) {
		panic("tuple is not a valid valuecommitment")
	}
	return x
}

func (vm *vm) pushValuecommitment(stacknum int, x valuecommitment) {
	vm.push(stacknum, x.entuple())
}

var assetcommitmentType = (*assetcommitment)(nil)

func (vm *vm) popAssetcommitment(stacknum int) assetcommitment {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x assetcommitment
	if !x.detuple(t) {
		panic("tuple is not a valid assetcommitment")
	}
	return x
}

func (vm *vm) pushAssetcommitment(stacknum int, x assetcommitment) {
	vm.push(stacknum, x.entuple())
}

var unprovenvalueType = (*unprovenvalue)(nil)

func (x unprovenvalue) entuple() tuple {
	return tuple{
		vbytes("unprovenvalue"),
		x.vc.entuple(),
	}
}

func (x *unprovenvalue) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "unprovenvalue" {
		return false
	}
	if !x.vc.detuple(t[1].(tuple)) {
		return false
	}
	return true
}

func (vm *vm) popUnprovenvalue(stacknum int) unprovenvalue {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x unprovenvalue
	if !x.detuple(t) {
		panic("tuple is not a valid unprovenvalue")
	}
	return x
}

func (vm *vm) pushUnprovenvalue(stacknum int, x unprovenvalue) {
	vm.push(stacknum, x.entuple())
}

var provenvalueType = (*provenvalue)(nil)

func (x provenvalue) entuple() tuple {
	return tuple{
		vbytes("provenvalue"),
		x.vc.entuple(),
		x.ac.entuple(),
	}
}

func (x *provenvalue) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "provenvalue" {
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

func (vm *vm) popProvenvalue(stacknum int) provenvalue {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x provenvalue
	if !x.detuple(t) {
		panic("tuple is not a valid provenvalue")
	}
	return x
}

func (vm *vm) pushProvenvalue(stacknum int, x provenvalue) {
	vm.push(stacknum, x.entuple())
}

var recordType = (*record)(nil)

func (x record) entuple() tuple {
	return tuple{
		vbytes("record"),
		vbytes(x.commandprogram),
		x.data,
	}
}

func (x *record) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "record" {
		return false
	}
	x.commandprogram = []byte(t[1].(vbytes))
	x.data = t[2]
	return true
}

func (vm *vm) popRecord(stacknum int) record {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x record
	if !x.detuple(t) {
		panic("tuple is not a valid record")
	}
	return x
}

func (vm *vm) pushRecord(stacknum int, x record) {
	vm.push(stacknum, x.entuple())
}

var inputType = (*input)(nil)

func (x input) entuple() tuple {
	return tuple{
		vbytes("input"),
		vbytes(x.contractid),
	}
}

func (x *input) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "input" {
		return false
	}
	x.contractid = []byte(t[1].(vbytes))
	return true
}

func (vm *vm) popInput(stacknum int) input {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x input
	if !x.detuple(t) {
		panic("tuple is not a valid input")
	}
	return x
}

func (vm *vm) pushInput(stacknum int, x input) {
	vm.push(stacknum, x.entuple())
}

var outputType = (*output)(nil)

func (x output) entuple() tuple {
	return tuple{
		vbytes("output"),
		vbytes(x.contractid),
	}
}

func (x *output) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "output" {
		return false
	}
	x.contractid = []byte(t[1].(vbytes))
	return true
}

func (vm *vm) popOutput(stacknum int) output {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x output
	if !x.detuple(t) {
		panic("tuple is not a valid output")
	}
	return x
}

func (vm *vm) pushOutput(stacknum int, x output) {
	vm.push(stacknum, x.entuple())
}

var readType = (*read)(nil)

func (x read) entuple() tuple {
	return tuple{
		vbytes("read"),
		vbytes(x.contractid),
	}
}

func (x *read) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "read" {
		return false
	}
	x.contractid = []byte(t[1].(vbytes))
	return true
}

func (vm *vm) popRead(stacknum int) read {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x read
	if !x.detuple(t) {
		panic("tuple is not a valid read")
	}
	return x
}

func (vm *vm) pushRead(stacknum int, x read) {
	vm.push(stacknum, x.entuple())
}

var programType = (*program)(nil)

func (x program) entuple() tuple {
	return tuple{
		vbytes("program"),
		vbytes(x.program),
	}
}

func (x *program) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "program" {
		return false
	}
	x.program = []byte(t[1].(vbytes))
	return true
}

func (vm *vm) popProgram(stacknum int) program {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x program
	if !x.detuple(t) {
		panic("tuple is not a valid program")
	}
	return x
}

func (vm *vm) pushProgram(stacknum int, x program) {
	vm.push(stacknum, x.entuple())
}

var nonceType = (*nonce)(nil)

func (x nonce) entuple() tuple {
	return tuple{
		vbytes("nonce"),
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
	if n, ok := t[0].(vbytes); !ok || string(n) != "nonce" {
		return false
	}
	x.program = []byte(t[1].(vbytes))
	x.mintime = int64(t[2].(vint64))
	x.maxtime = int64(t[3].(vint64))
	x.blockchainid = []byte(t[4].(vbytes))
	return true
}

func (vm *vm) popNonce(stacknum int) nonce {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x nonce
	if !x.detuple(t) {
		panic("tuple is not a valid nonce")
	}
	return x
}

func (vm *vm) pushNonce(stacknum int, x nonce) {
	vm.push(stacknum, x.entuple())
}

var assetdefinitionType = (*assetdefinition)(nil)

func (x assetdefinition) entuple() tuple {
	return tuple{
		vbytes("assetdefinition"),
		x.issuanceprogram.entuple(),
	}
}

func (x *assetdefinition) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "assetdefinition" {
		return false
	}
	if !x.issuanceprogram.detuple(t[1].(tuple)) {
		return false
	}
	return true
}

func (vm *vm) popAssetdefinition(stacknum int) assetdefinition {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x assetdefinition
	if !x.detuple(t) {
		panic("tuple is not a valid assetdefinition")
	}
	return x
}

func (vm *vm) pushAssetdefinition(stacknum int, x assetdefinition) {
	vm.push(stacknum, x.entuple())
}

var issuancecandidateType = (*issuancecandidate)(nil)

func (x issuancecandidate) entuple() tuple {
	return tuple{
		vbytes("issuancecandidate"),
		vbytes(x.assetID),
		vbytes(x.issuanceKey),
	}
}

func (x *issuancecandidate) detuple(t tuple) bool {
	if len(t) != 3 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "issuancecandidate" {
		return false
	}
	x.assetID = []byte(t[1].(vbytes))
	x.issuanceKey = []byte(t[2].(vbytes))
	return true
}

func (vm *vm) popIssuancecandidate(stacknum int) issuancecandidate {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x issuancecandidate
	if !x.detuple(t) {
		panic("tuple is not a valid issuancecandidate")
	}
	return x
}

func (vm *vm) pushIssuancecandidate(stacknum int, x issuancecandidate) {
	vm.push(stacknum, x.entuple())
}

var maxtimeType = (*maxtime)(nil)

func (x maxtime) entuple() tuple {
	return tuple{
		vbytes("maxtime"),
		vint64(x.maxtime),
	}
}

func (x *maxtime) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "maxtime" {
		return false
	}
	x.maxtime = int64(t[1].(vint64))
	return true
}

func (vm *vm) popMaxtime(stacknum int) maxtime {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x maxtime
	if !x.detuple(t) {
		panic("tuple is not a valid maxtime")
	}
	return x
}

func (vm *vm) pushMaxtime(stacknum int, x maxtime) {
	vm.push(stacknum, x.entuple())
}

var mintimeType = (*mintime)(nil)

func (x mintime) entuple() tuple {
	return tuple{
		vbytes("mintime"),
		vint64(x.mintime),
	}
}

func (x *mintime) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "mintime" {
		return false
	}
	x.mintime = int64(t[1].(vint64))
	return true
}

func (vm *vm) popMintime(stacknum int) mintime {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x mintime
	if !x.detuple(t) {
		panic("tuple is not a valid mintime")
	}
	return x
}

func (vm *vm) pushMintime(stacknum int, x mintime) {
	vm.push(stacknum, x.entuple())
}

var annotationType = (*annotation)(nil)

func (x annotation) entuple() tuple {
	return tuple{
		vbytes("annotation"),
		vbytes(x.data),
	}
}

func (x *annotation) detuple(t tuple) bool {
	if len(t) != 2 {
		return false
	}
	if n, ok := t[0].(vbytes); !ok || string(n) != "annotation" {
		return false
	}
	x.data = []byte(t[1].(vbytes))
	return true
}

func (vm *vm) popAnnotation(stacknum int) annotation {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x annotation
	if !x.detuple(t) {
		panic("tuple is not a valid annotation")
	}
	return x
}

func (vm *vm) pushAnnotation(stacknum int, x annotation) {
	vm.push(stacknum, x.entuple())
}

var legacyoutputType = (*legacyoutput)(nil)

func (x legacyoutput) entuple() tuple {
	return tuple{
		vbytes("legacyoutput"),
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
	if n, ok := t[0].(vbytes); !ok || string(n) != "legacyoutput" {
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

func (vm *vm) popLegacyoutput(stacknum int) legacyoutput {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x legacyoutput
	if !x.detuple(t) {
		panic("tuple is not a valid legacyoutput")
	}
	return x
}

func (vm *vm) pushLegacyoutput(stacknum int, x legacyoutput) {
	vm.push(stacknum, x.entuple())
}

var vm1programType = (*vm1program)(nil)

func (x vm1program) entuple() tuple {
	return tuple{
		vbytes("vm1program"),
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
	if n, ok := t[0].(vbytes); !ok || string(n) != "vm1program" {
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

func (vm *vm) popVm1program(stacknum int) vm1program {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var x vm1program
	if !x.detuple(t) {
		panic("tuple is not a valid vm1program")
	}
	return x
}

func (vm *vm) pushVm1program(stacknum int, x vm1program) {
	vm.push(stacknum, x.entuple())
}
