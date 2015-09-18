package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"golang.org/x/net/context"

	"chain/net/http/pat"
)

// DefaultResponse will be sent as the response body
// when the handler function signature
// has no return value.
var DefaultResponse = json.RawMessage(`{"message":"ok"}`)

// handler is an http.Handler that calls a function for each request.
// It uses the signature of the function to decide how to interpret
type handler struct {
	fv      reflect.Value
	labels  []string
	inType  reflect.Type
	hasCtx  bool
	errFunc ErrorWriter
}

func newHandler(pattern string, f interface{}, errFunc ErrorWriter) (*handler, error) {
	fv := reflect.ValueOf(f)
	labels := pat.Labels(pattern)
	hasCtx, inType, err := funcInputType(fv, len(labels))
	if err != nil {
		return nil, err
	}

	h := &handler{fv, labels, inType, hasCtx, errFunc}
	return h, nil
}

func (h *handler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var a []reflect.Value
	if h.hasCtx {
		ctx = context.WithValue(ctx, reqKey, req)
		ctx = context.WithValue(ctx, respKey, w)
		a = append(a, reflect.ValueOf(ctx))
	}
	for _, label := range h.labels {
		s := req.URL.Query().Get(label)
		a = append(a, reflect.ValueOf(s))
	}
	if h.inType != nil {
		inPtr := reflect.New(h.inType)
		err := Read(ctx, req.Body, inPtr.Interface())
		if err != nil {
			h.errFunc(ctx, w, err)
			return
		}
		a = append(a, inPtr.Elem())
	}
	rv := h.fv.Call(a)

	var (
		res interface{}
		err error
	)
	switch n := len(rv); {
	case n == 0:
		res = &DefaultResponse
	case n == 1 && !h.fv.Type().Out(0).Implements(errorType):
		res = rv[0].Interface()
	case n == 1 && h.fv.Type().Out(0).Implements(errorType):
		// out param is of type error; its value can still be nil
		res = &DefaultResponse
		err, _ = rv[0].Interface().(error)
	case n == 2:
		res = rv[0].Interface()
		err, _ = rv[1].Interface().(error)
	}
	if err != nil {
		h.errFunc(ctx, w, err)
		return
	}

	Write(ctx, w, 200, res)
}

var (
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
)

func funcInputType(fv reflect.Value, nlabel int) (hasCtx bool, t reflect.Type, err error) {
	ft := fv.Type()
	if ft.Kind() != reflect.Func || ft.IsVariadic() {
		return false, nil, errors.New("need nonvariadic func in " + ft.String())
	}

	off := 0 // or 1 with context
	hasCtx = ft.NumIn() >= 1 && ft.In(0).Implements(contextType)
	if hasCtx {
		off = 1
	}

	if ft.NumIn() < off+nlabel {
		return false, nil, errors.New("not enough params in " + ft.String())
	} else if ft.NumIn() > off+nlabel+1 {
		return false, nil, errors.New("too many params in " + ft.String())
	}
	for i := 0; i < nlabel; i++ {
		if ft.In(off+i).Kind() != reflect.String {
			return false, nil, fmt.Errorf("param %d must be string label in %s", i, ft)
		}
	}

	if ft.NumIn() == off+nlabel+1 {
		t = ft.In(ft.NumIn() - 1)
	}

	if n := ft.NumOut(); n == 2 && !ft.Out(1).Implements(errorType) {
		return false, nil, errors.New("second return value must be error in " + ft.String())
	} else if n > 2 {
		return false, nil, errors.New("need at most two return values in " + ft.String())
	}

	return hasCtx, t, nil
}
