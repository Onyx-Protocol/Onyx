package pat

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/context"

	chainhttp "chain/net/http"
)

func TestPatMatch(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    url.Values
	}{
		{"/", "/", url.Values{}},
		{"/", "/wrong_url", url.Values{}},
		{"/foo/:name", "/foo/bar", url.Values{":name": {"bar"}}},
		{"/foo/:name/baz", "/foo/bar", nil},
		{"/foo/:name/bar/", "/foo/keith/bar/baz", url.Values{":name": {"keith"}}},
		{"/foo/:name/bar/", "/foo/keith/bar/", url.Values{":name": {"keith"}}},
		{"/foo/:name/bar/", "/foo/keith/bar", nil},
		{"/foo/:name/baz", "/foo/bar/baz", url.Values{":name": {"bar"}}},
		{"/foo/:name/baz/:id", "/foo/bar/baz", nil},
		{"/foo/:name/baz/:id", "/foo/bar/baz/123", url.Values{":name": {"bar"}, ":id": {"123"}}},
		{"/foo/:name/baz/:name", "/foo/bar/baz/123", url.Values{":name": {"bar", "123"}}},
		{"/foo/:name.txt", "/foo/bar.txt", url.Values{":name": {"bar"}}},
		{"/foo/:name", "/foo/:bar", url.Values{":name": {":bar"}}},
		{"/foo/:a:b", "/foo/val1:val2", url.Values{":a": {"val1"}, ":b": {":val2"}}},
		{"/foo/:a.", "/foo/.", url.Values{":a": {""}}},
		{"/foo/:a:b", "/foo/:bar", url.Values{":a": {""}, ":b": {":bar"}}},
		{"/foo/:a:b:c", "/foo/:bar", url.Values{":a": {""}, ":b": {""}, ":c": {":bar"}}},
		{"/foo/::name", "/foo/val1:val2", url.Values{":": {"val1"}, ":name": {":val2"}}},
		{"/foo/:name.txt", "/foo/bar/baz.txt", nil},
		{"/foo/x:name", "/foo/bar", nil},
		{"/foo/x:name", "/foo/xbar", url.Values{":name": {"bar"}}},
	}

	for _, test := range cases {
		h := &patHandler{test.pattern, nil}
		got, _ := h.try(test.path)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("try(%q, %q) = %v want %v", test.pattern, test.path, got, test.want)
		}
	}
}

func TestPatRoutingHit(t *testing.T) {
	p := New()

	var ok bool
	p.Get("/foo/:name", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok = true
		t.Logf("%#v", r.URL.Query())
		if got := r.URL.Query().Get(":name"); got != "keith" {
			t.Fatalf("got = %q want keith", got)
		}
	}))

	r, err := http.NewRequest("GET", "/foo/keith?a=b", nil)
	if err != nil {
		t.Fatal(err)
	}

	p.ServeHTTPContext(context.Background(), nil, r)

	if !ok {
		t.Fail()
	}
}

func TestPatRoutingMethodNotAllowed(t *testing.T) {
	p := New()

	var ok bool
	p.Post("/foo/:name", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok = true
	}))

	p.Put("/foo/:name", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok = true
	}))

	r, err := http.NewRequest("GET", "/foo/keith", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	p.ServeHTTPContext(context.Background(), rr, r)

	if ok {
		t.Fail()
	}
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}

	got := strings.Split(rr.Header().Get("Allow"), ", ")
	sort.Strings(got)
	want := []string{"POST", "PUT"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Allow = %v want %v", got, want)
	}
}

// Check to make sure we don't pollute the Raw Query when we have no parameters
func TestPatNoParams(t *testing.T) {
	p := New()

	var ok bool
	p.Get("/foo/", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok = true
		t.Logf("%#v", r.URL.RawQuery)
		want := ""
		got := r.URL.RawQuery
		if got != want {
			t.Fatalf("got = %q want %q", got, want)
		}
	}))

	r, err := http.NewRequest("GET", "/foo/", nil)
	if err != nil {
		t.Fatal(err)
	}

	p.ServeHTTPContext(context.Background(), nil, r)

	if !ok {
		t.Fail()
	}
}

// Check to make sure we don't pollute the Raw Query when there are parameters but no pattern variables
func TestPatOnlyUserParams(t *testing.T) {
	p := New()

	var ok bool
	p.Get("/foo/", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok = true
		t.Logf("%#v", r.URL.RawQuery)
		want := "a=b"
		got := r.URL.RawQuery
		if got != want {
			t.Fatalf("got = %q want %q", got, want)
		}
	}))

	r, err := http.NewRequest("GET", "/foo/?a=b", nil)
	if err != nil {
		t.Fatal(err)
	}

	p.ServeHTTPContext(context.Background(), nil, r)

	if !ok {
		t.Fail()
	}
}

func TestPatImplicitRedirect(t *testing.T) {
	p := New()
	p.Get("/foo/", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {}))

	r, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	p.ServeHTTPContext(context.Background(), res, r)

	if res.Code != 301 {
		t.Errorf("expected Code 301, was %d", res.Code)
	}

	if loc := res.Header().Get("Location"); loc != "/foo/" {
		t.Errorf("expected %q, got %q", "/foo/", loc)
	}

	p = New()
	p.Get("/foo", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {}))
	p.Get("/foo/", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {}))

	r, err = http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	res = httptest.NewRecorder()
	res.Code = 200
	p.ServeHTTPContext(context.Background(), res, r)

	if res.Code != 200 {
		t.Errorf("expected Code 200, was %d", res.Code)
	}
}

func TestTail(t *testing.T) {
	for i, test := range []struct {
		pat    string
		path   string
		expect string
	}{
		{"/:a/", "/x/y/z", "y/z"},
		{"/:a/", "/x", ""},
		{"/:a/", "/x/", ""},
		{"/:a", "/x/y/z", ""},
		{"/b/:a", "/x/y/z", ""},
		{"/hello/:title/", "/hello/mr/mizerany", "mizerany"},
		{"/:a/", "/x/y/z", "y/z"},
	} {
		tail := Tail(test.pat, test.path)
		if tail != test.expect {
			t.Errorf("failed test %d: Tail(%q, %q) == %q (!= %q)",
				i, test.pat, test.path, tail, test.expect)
		}
	}
}

func TestLabels(t *testing.T) {
	cases := []struct{ pattern, want string }{
		{"/", ""},
		{"/foo/:name", ":name"},
		{"/foo/:name/baz", ":name"},
		{"/foo/:name/bar/", ":name"},
		{"/foo/:name/baz/:id", ":name :id"},
		{"/foo/:name/baz/:name", ":name :name"},
		{"/foo/:name.txt", ":name"},
		{"/foo/:name", ":name"},
		{"/foo/:a:b", ":a :b"},
		{"/foo/:a.", ":a"},
		{"/foo/:a:b", ":a :b"},
		{"/foo/:a:b:c", ":a :b :c"},
		{"/foo/::name", ": :name"},
		{"/foo/:name.txt", ":name"},
		{"/foo/x:name", ":name"},
		{"/:a/", ":a"},
		{"/:a", ":a"},
		{"/b/:a", ":a"},
		{"/hello/:title/", ":title"},
	}

	for _, test := range cases {
		got := strings.Join(Labels(test.pattern), " ")
		if got != test.want {
			t.Errorf("Labels(%q) = %q want %q", test.pattern, got, test.want)
		}
	}
}

func TestLongestMatch(t *testing.T) {
	p := New()
	var ok bool
	p.Get("/foo/:variable", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		t.Fatal("failed test: shorter pattern was matched")
	}))
	p.Get("/foo/longerpath", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ok = true
	}))

	r, err := http.NewRequest("GET", "/foo/longerpath", nil)
	if err != nil {
		t.Fatal(err)
	}

	p.ServeHTTPContext(context.Background(), httptest.NewRecorder(), r)
	if !ok {
		t.Fail()
	}
}
