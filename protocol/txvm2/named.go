package txvm2

const (
	annotationTuple    = "annotation"
	valueTuple         = "value"
	transactionIDTuple = "transactionID"
)

var namedTuples = map[string][]int{
	annotationTuple:    []int{stringtype},
	transactionIDTuple: []int{stringtype},
	valueTuple:         []int{int64type, stringtype},
}

func (t tuple) name() (string, bool) {
	if len(t) == 0 {
		return "", false
	}
	b, ok := t[0].(vstring)
	if !ok {
		return "", false
	}
	return string(b), true
}

func mkAnnotation(data vstring) tuple {
	return tuple{
		vstring(annotationTuple),
		data,
	}
}
