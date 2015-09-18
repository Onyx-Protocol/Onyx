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
		pattern  string
		rawQuery string
		input    string
		output   string
		f        interface{}
		wantErr  error
	}{
		{"/", "", ``, `{"message":"ok"}`, func() {}, nil},
		{"/", "", ``, `1`, func() int { return 1 }, nil},
		{"/", "", ``, `{"message":"ok"}`, func() error { return nil }, nil},
		{"/", "", ``, ``, func() error { return errX }, errX},
		{"/", "", ``, `1`, func() (int, error) { return 1, nil }, nil},
		{"/", "", ``, ``, func() (int, error) { return 0, errX }, errX},
		{"/", "", `1`, `1`, func(i int) int { return i }, nil},
		{"/", "", `1`, `1`, func(i *int) int { return *i }, nil},
		{"/:a", ":a=foo", ``, `"foo"`, func(s string) string { return s }, nil},
		{"/", "", `"foo"`, `"foo"`, func(s string) string { return s }, nil},
		{"/", "", `{"x":1}`, `1`, func(x struct{ X int }) int { return x.X }, nil},
		{"/", "", `{"x":1}`, `1`, func(x *struct{ X int }) int { return x.X }, nil},
		{"/", "", ``, `1`, func(ctx context.Context) int { return ctx.Value("k").(int) }, nil},
	}

	for _, test := range cases {
		var gotErr error
		errFunc := func(ctx context.Context, w http.ResponseWriter, err error) {
			gotErr = err
		}
		h, err := newHandler(test.pattern, test.f, errFunc)
		if err != nil {
			t.Errorf("NewHandler(%q, %v) got err %v", test.pattern, test.f, err)
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
	h, _ := newHandler("/", func(int) {}, errFunc)

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
	cases := []struct {
		nlabel int
		pat    string
		f      interface{}
	}{
		{0, "/", 0},
		{0, "/", "foo"},
		{0, "/", func() (int, int) { return 0, 0 }},
		{1, "/:n", func() {}},
		{1, "/:n", func(int) {}},
		{0, "/", func(string, int) {}},
		{0, "/", func() (int, int, error) { return 0, 0, nil }},
	}

	for _, test := range cases {
		_, _, err := funcInputType(reflect.ValueOf(test.f), test.nlabel)
		if err == nil {
			t.Errorf("funcInputType(%T, %d) want error", test.f, test.nlabel)
		}

		_, err = newHandler(test.pat, test.f, nil)
		if err == nil {
			t.Errorf("funcInputType(%T, %d) want error", test.f, test.nlabel)
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
		nlabel  int
		f       interface{}
		wantCtx bool
		wantT   reflect.Type
	}{
		{0, func() {}, false, nil},
		{0, func() int { return 0 }, false, nil},
		{0, func() error { return nil }, false, nil},
		{0, func() (int, error) { return 0, nil }, false, nil},
		{0, func(int) {}, false, intType},
		{0, func(*int) {}, false, intpType},
		{0, func(context.Context) {}, true, nil},
		{0, func(string) {}, false, stringType}, // req body is string
		{1, func(string) {}, false, nil},        // one label; no req body
		{1, func(label, body string) {}, false, stringType},
	}

	for _, test := range cases {
		gotCtx, gotT, err := funcInputType(reflect.ValueOf(test.f), test.nlabel)
		if err != nil {
			t.Errorf("funcInputType(%T, %d) got error: %v", test.f, test.nlabel, err)
		}
		if gotCtx != test.wantCtx {
			t.Errorf("funcInputType(%T, %d) context = %v want %v", test.f, test.nlabel, gotCtx, test.wantCtx)
		}
		if gotT != test.wantT {
			t.Errorf("funcInputType(%T, %d) = %v want %v", test.f, test.nlabel, gotT, test.wantT)
		}
	}
}
