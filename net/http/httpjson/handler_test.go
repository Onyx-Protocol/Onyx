package httpjson

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"golang.org/x/net/context"

	"chain/errors"
)

func TestHandler(t *testing.T) {
	errX := errors.New("x")

	cases := []struct {
		rawQuery string
		input    string
		output   string
		f        interface{}
		wantErr  error
	}{
		{"", ``, `{"message":"ok"}`, func() {}, nil},
		{"", ``, `1`, func() int { return 1 }, nil},
		{"", ``, `{"message":"ok"}`, func() error { return nil }, nil},
		{"", ``, ``, func() error { return errX }, errX},
		{"", ``, `1`, func() (int, error) { return 1, nil }, nil},
		{"", ``, ``, func() (int, error) { return 0, errX }, errX},
		{"", `1`, `1`, func(i int) int { return i }, nil},
		{"", `1`, `1`, func(i *int) int { return *i }, nil},
		{"", `"foo"`, `"foo"`, func(s string) string { return s }, nil},
		{"", `{"x":1}`, `1`, func(x struct{ X int }) int { return x.X }, nil},
		{"", `{"x":1}`, `1`, func(x *struct{ X int }) int { return x.X }, nil},
		{"", ``, `1`, func(ctx context.Context) int { return ctx.Value("k").(int) }, nil},
	}

	for _, test := range cases {
		var gotErr error
		errFunc := func(ctx context.Context, w http.ResponseWriter, err error) {
			gotErr = err
		}
		h, err := Handler(test.f, errFunc)
		if err != nil {
			t.Errorf("Handler(%v) got err %v", test.f, err)
			continue
		}

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", strings.NewReader(test.input))
		req.URL.RawQuery = test.rawQuery
		ctx := context.WithValue(context.Background(), "k", 1)
		h.ServeHTTPContext(ctx, resp, req)
		if resp.Code != 200 {
			t.Errorf("%T response code = %d want 200", test.f, resp.Code)
		}
		got := strings.TrimSpace(resp.Body.String())
		if got != test.output {
			t.Errorf("%T response body = %#q want %#q", test.f, got, test.output)
		}
		if gotErr != test.wantErr {
			t.Errorf("%T err = %v want %v", test.f, gotErr, test.wantErr)
		}
	}
}

func TestReadErr(t *testing.T) {
	var gotErr error
	errFunc := func(ctx context.Context, w http.ResponseWriter, err error) {
		gotErr = errors.Root(err)
	}
	h, _ := Handler(func(int) {}, errFunc)

	resp := httptest.NewRecorder()
	body := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("123456")))
	req, _ := http.NewRequest("GET", "/", body)
	h.ServeHTTPContext(nil, resp, req)
	if got := resp.Body.Len(); got != 0 {
		t.Errorf("len(response) = %d want 0", got)
	}
	wantErr := ErrBadRequest
	if gotErr != wantErr {
		t.Errorf("err = %v want %v", gotErr, wantErr)
	}
}

func TestFuncInputTypeError(t *testing.T) {
	cases := []interface{}{
		0,
		"foo",
		func() (int, int) { return 0, 0 },
		func(string, int) {},
		func() (int, int, error) { return 0, 0, nil },
	}

	for _, testf := range cases {
		_, _, err := funcInputType(reflect.ValueOf(testf))
		if err == nil {
			t.Errorf("funcInputType(%T) want error", testf)
		}

		_, err = Handler(testf, nil)
		if err == nil {
			t.Errorf("funcInputType(%T) want error", testf)
		}
	}
}

var (
	intType    = reflect.TypeOf(0)
	intpType   = reflect.TypeOf((*int)(nil))
	stringType = reflect.TypeOf("")
)

func TestFuncInputTypeOk(t *testing.T) {
	cases := []struct {
		f       interface{}
		wantCtx bool
		wantT   reflect.Type
	}{
		{func() {}, false, nil},
		{func() int { return 0 }, false, nil},
		{func() error { return nil }, false, nil},
		{func() (int, error) { return 0, nil }, false, nil},
		{func(int) {}, false, intType},
		{func(*int) {}, false, intpType},
		{func(context.Context) {}, true, nil},
		{func(string) {}, false, stringType}, // req body is string
	}

	for _, test := range cases {
		gotCtx, gotT, err := funcInputType(reflect.ValueOf(test.f))
		if err != nil {
			t.Errorf("funcInputType(%T) got error: %v", test.f, err)
		}
		if gotCtx != test.wantCtx {
			t.Errorf("funcInputType(%T) context = %v want %v", test.f, gotCtx, test.wantCtx)
		}
		if gotT != test.wantT {
			t.Errorf("funcInputType(%T) = %v want %v", test.f, gotT, test.wantT)
		}
	}
}
