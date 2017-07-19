package txvm2

// TODO: define typesafe structs for these named tuple types and
// codegen the name mappings and encoding to/from txvm tuples

const (
	anchorTuple          = "anchor"
	annotationTuple      = "annotation"
	assetCommitmentTuple = "assetcommitment"
	assetDefinitionTuple = "assetdefinition"
	contractTuple        = "contract"
	inputTuple           = "input"
	legacyOutputTuple    = "legacyoutput"
	maxtimeTuple         = "maxtime"
	mintimeTuple         = "mintime"
	nonceTuple           = "nonce"
	outputTuple          = "output"
	programTuple         = "program"
	provenValueTuple     = "provenvalue"
	readTuple            = "read"
	recordTuple          = "record"
	retirementTuple      = "retirement"
	transactionIDTuple   = "transactionID"
	transactionTuple     = "tx"
	unprovenValueTuple   = "unprovenvalue"
	valueCommitmentTuple = "valuecommitment"
	valueTuple           = "value"
)

var namedTuples = map[string][]int{
	anchorTuple:          {bytestype},
	annotationTuple:      {bytestype},
	assetCommitmentTuple: {bytestype, bytestype},
	assetDefinitionTuple: {bytestype},
	contractTuple:        {tupletype, bytestype, bytestype}, // TODO: be more specific about the field types
	inputTuple:           {bytestype},
	legacyOutputTuple:    {bytestype, bytestype, int64type, int64type, bytestype, bytestype},
	maxtimeTuple:         {int64type},
	mintimeTuple:         {int64type},
	nonceTuple:           {bytestype, int64type, int64type, bytestype},
	outputTuple:          {bytestype},
	programTuple:         {bytestype},
	provenValueTuple:     {tupletype, tupletype},
	readTuple:            {bytestype},
	recordTuple:          {bytestype, 0}, // 0 means "any value"
	retirementTuple:      {},             // xxx
	transactionTuple:     {int64type, int64type, bytestype},
	transactionIDTuple:   {bytestype},
	unprovenValueTuple:   {tupletype},
	valueCommitmentTuple: {bytestype, bytestype},
	valueTuple:           {int64type, bytestype},
}

func (t tuple) name() (string, bool) {
	if len(t) == 0 {
		return "", false
	}
	b, ok := t[0].(vbytes)
	if !ok {
		return "", false
	}
	return string(b), true
}

func isNamed(v item, s string) bool {
	t, ok := v.(tuple)
	if !ok {
		return false
	}
	n, ok := t.name()
	if !ok {
		return false
	}
	if s != n {
		return false
	}
	if len(t) != len(namedTuples[n])+1 {
		return false
	}
	for i, typ := range namedTuples[n] {
		if typ == 0 {
			continue
		}
		if t[i+1].typ() != typ {
			return false
		}
	}
	return true
}

func mkAnchor(val vbytes) tuple {
	return tuple{vbytes(anchorTuple), val}
}

func mkAnnotation(data vbytes) tuple {
	return tuple{vbytes(annotationTuple), data}
}

func mkAssetCommitment(h, c vbytes) tuple {
	return tuple{vbytes(assetCommitmentTuple), h, c}
}

func mkAssetDefinition(prog vbytes) tuple {
	return tuple{vbytes(assetDefinitionTuple), prog}
}

func mkContract(val tuple, prog, anchor vbytes) tuple {
	return tuple{vbytes(contractTuple), val, prog, anchor}
}

func mkInput(contractID vbytes) tuple {
	return tuple{vbytes(inputTuple), contractID}
}

func mkMaxtime(max vint64) tuple {
	return tuple{vbytes(maxtimeTuple), max}
}

func mkMintime(min vint64) tuple {
	return tuple{vbytes(mintimeTuple), min}
}

func mkNonce(prog vbytes, min, max vint64, bcID vbytes) tuple {
	return tuple{vbytes(nonceTuple), prog, min, max, bcID}
}

func mkOutput(contractID vbytes) tuple {
	return tuple{vbytes(outputTuple), contractID}
}

func mkProgram(prog vbytes) tuple {
	return tuple{vbytes(programTuple), prog}
}

func mkRead(contractID vbytes) tuple {
	return tuple{vbytes(readTuple), contractID}
}

func mkRecord(prog vbytes, data item) tuple {
	return tuple{vbytes(recordTuple), prog, data}
}

func mkRetirement(val tuple) tuple {
	return tuple{} // xxx
}

func mkTransaction(version, runlimit vint64, effectHash vbytes) tuple {
	return tuple{vbytes(transactionTuple), version, runlimit, effectHash}
}

func mkUnprovenValue(vc tuple) tuple {
	return tuple{vbytes(unprovenValueTuple), vc}
}

func mkValue(amount vint64, assetID vbytes) tuple {
	return tuple{vbytes(valueTuple), amount, assetID}
}

func mkValueCommitment(v, f vbytes) tuple {
	return tuple{vbytes(valueCommitmentTuple), v, f}
}

func anchorValue(anchor tuple) vbytes {
	return anchor[1].(vbytes)
}

func assetCommitmentAssetPoint(ac tuple) vbytes {
	return ac[1].(vbytes)
}

func assetCommitmentBlindingPoint(ac tuple) vbytes {
	return ac[2].(vbytes)
}

func legacyOutputAmount(out tuple) vint64 {
	return out[3].(vint64) // xxx if legacy outputs have no type string, this is off by one
}

func legacyOutputAssetID(out tuple) vbytes {
	return out[2].(vbytes) // xxx if legacy outputs have no type string, this is off by one
}

func legacyOutputData(out tuple) vbytes {
	return out[6].(vbytes) // xxx if legacy outputs have no type string, this is off by one
}

func legacyOutputIndex(out tuple) vint64 {
	return out[4].(vint64) // xxx if legacy outputs have no type string, this is off by one
}

func legacyOutputProgram(out tuple) vbytes {
	return out[5].(vbytes) // xxx if legacy outputs have no type string, this is off by one
}

func legacyOutputSourceID(out tuple) vbytes {
	return out[1].(vbytes) // xxx if legacy outputs have no type string, this is off by one
}

func programProgram(prog tuple) vbytes {
	return prog[1].(vbytes)
}

func provenValueAssetCommitment(pv tuple) tuple {
	return pv[2].(tuple)
}

func provenValueValueCommitment(pv tuple) tuple {
	return pv[1].(tuple)
}

func recordCommandProgram(rec tuple) vbytes {
	return rec[1].(vbytes)
}

func unprovenValueValueCommitment(pv tuple) tuple {
	return pv[1].(tuple)
}

func valueAmount(val tuple) vint64 {
	return val[1].(vint64)
}

func valueAssetID(val tuple) vbytes {
	return val[2].(vbytes)
}

func valueCommitmentBlindingPoint(vc tuple) vbytes {
	return vc[2].(vbytes)
}

func valueCommitmentValuePoint(vc tuple) vbytes {
	return vc[1].(vbytes)
}
