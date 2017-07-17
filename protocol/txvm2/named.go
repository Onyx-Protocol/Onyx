package txvm2

const (
	annotationTuple    = "annotation"
	valueTuple         = "value"
	transactionIDTuple = "transactionID"
	programTuple       = "program"
)

var namedTuples = map[string][]int{
	annotationTuple:    []int{bytestype},
	transactionIDTuple: []int{bytestype},
	valueTuple:         []int{int64type, bytestype},
	programTuple:       []int{bytestype},
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

func isNamed(v value, s string) bool {
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
		if t[i+1].typ() != typ {
			return false
		}
	}
	return true
}

func mkAnnotation(data vbytes) tuple {
	return tuple{
		vbytes(annotationTuple),
		data,
	}
}
