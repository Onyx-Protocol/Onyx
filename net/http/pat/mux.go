// Package pat implements a simple URL pattern muxer
package pat

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/context"

	chainhttp "chain/net/http"
)

// PatternServeMux is an HTTP request multiplexer. It matches the URL of each
// incoming request against a list of registered patterns with their associated
// methods and calls the handler for the pattern that most closely matches the
// URL.
//
// Pattern matching attempts each pattern in the order in which they were
// registered and selects the longest matching pattern.
//
// Patterns may contain literals or captures. Capture names start with a colon
// and consist of letters A-Z, a-z, _, and 0-9. The rest of the pattern
// matches literally. The portion of the URL matching each name ends with an
// occurrence of the character in the pattern immediately following the name,
// or a /, whichever comes first. It is possible for a name to match the empty
// string.
//
// Example pattern with one capture:
//   /hello/:name
// Will match:
//   /hello/blake
//   /hello/keith
// Will not match:
//   /hello/blake/
//   /hello/blake/foo
//   /foo
//   /foo/bar
//
// Example 2:
//    /hello/:name/
// Will match:
//   /hello/blake/
//   /hello/keith/foo
//   /hello/blake
//   /hello/keith
// Will not match:
//   /foo
//   /foo/bar
//
// A pattern ending with a slash will get an implicit redirect to it's
// non-slash version.  For example: Get("/foo/", handler) will implicitly
// register Get("/foo", handler). You may override it by registering
// Get("/foo", anotherhandler) before the slash version.
//
// Retrieve the capture from the r.URL.Query().Get(":name") in a handler (note
// the colon). If a capture name appears more than once, the additional values
// are appended to the previous values (see
// http://golang.org/pkg/net/url/#Values)
//
// A trivial example server is:
//
//	package main
//
//	import (
//		"io"
//		"net/http"
//		"log"
//
//		"chain/net/http/pat"
//	)
//
//	// hello world, the web server
//	func HelloServer(w http.ResponseWriter, req *http.Request) {
//		io.WriteString(w, "hello, "+req.URL.Query().Get(":name")+"!\n")
//	}
//
//	func main() {
//		m := pat.New()
//		m.Get("/hello/:name", http.HandlerFunc(HelloServer))
//
//		// Register this pat with the default serve mux so that other packages
//		// may also be exported. (i.e. /debug/pprof/*)
//		http.Handle("/", m)
//		err := http.ListenAndServe(":12345", nil)
//		if err != nil {
//			log.Fatal("ListenAndServe: ", err)
//		}
//	}
//
// When "Method Not Allowed":
//
// Pat knows what methods are allowed given a pattern and a URI. For
// convenience, PatternServeMux will add the Allow header for requests that
// match a pattern for a method other than the method requested and set the
// Status to "405 Method Not Allowed".
type PatternServeMux struct {
	handlers map[string][]*patHandler
}

// New returns a new PatternServeMux.
func New() *PatternServeMux {
	return &PatternServeMux{make(map[string][]*patHandler)}
}

// ServeHTTPContext r.URL.Path against its routing table using the rules
// described above.
func (p *PatternServeMux) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h := p.Handler(r)
	h.ServeHTTPContext(ctx, w, r)
}

// Handler edits r's query parameters to include elements from the path.
func (p *PatternServeMux) Handler(r *http.Request) chainhttp.Handler {
	var h chainhttp.Handler
	var b int
	var par url.Values
	for _, ph := range p.handlers[r.Method] {
		if params, ok := ph.try(r.URL.Path); ok {
			if len(ph.pat) > b {
				h = ph.h
				b = len(ph.pat)
				par = params
			}
		}
	}
	if h != nil {
		if len(par) > 0 {
			r.URL.RawQuery = url.Values(par).Encode() + "&" + r.URL.RawQuery
		}
		return h
	}

	allowed := make([]string, 0, len(p.handlers))
	for meth, handlers := range p.handlers {
		if meth == r.Method {
			continue
		}

		for _, ph := range handlers {
			if _, ok := ph.try(r.URL.Path); ok {
				allowed = append(allowed, meth)
			}
		}
	}

	if len(allowed) == 0 {
		return chainhttp.DropContext(http.NotFoundHandler())
	}

	return chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Allow", strings.Join(allowed, ", "))
		http.Error(w, `{"message": "Method not allowed"}`, 405)
	})
}

// Head will register a pattern with a handler for HEAD requests.
func (p *PatternServeMux) Head(pat string, h chainhttp.Handler) {
	p.Add("HEAD", pat, h)
}

// Get will register a pattern with a handler for GET requests.
// It also registers pat for HEAD requests. If this needs to be overridden, use
// Head before Get with pat.
func (p *PatternServeMux) Get(pat string, h chainhttp.Handler) {
	p.Add("HEAD", pat, h)
	p.Add("GET", pat, h)
}

// Post will register a pattern with a handler for POST requests.
func (p *PatternServeMux) Post(pat string, h chainhttp.Handler) {
	p.Add("POST", pat, h)
}

// Put will register a pattern with a handler for PUT requests.
func (p *PatternServeMux) Put(pat string, h chainhttp.Handler) {
	p.Add("PUT", pat, h)
}

// Del will register a pattern with a handler for DELETE requests.
func (p *PatternServeMux) Del(pat string, h chainhttp.Handler) {
	p.Add("DELETE", pat, h)
}

// Options will register a pattern with a handler for OPTIONS requests.
func (p *PatternServeMux) Options(pat string, h chainhttp.Handler) {
	p.Add("OPTIONS", pat, h)
}

// Add will register a pattern with a handler for meth requests.
func (p *PatternServeMux) Add(meth, pat string, h chainhttp.Handler) {
	p.handlers[meth] = append(p.handlers[meth], &patHandler{pat, h})

	n := len(pat)
	if n > 0 && pat[n-1] == '/' {
		p.Add(meth, pat[:n-1], chainhttp.DropContext(http.RedirectHandler(pat, http.StatusMovedPermanently)))
	}
}

func (p *PatternServeMux) AddFunc(meth, pat string, f chainhttp.HandlerFunc) {
	p.Add(meth, pat, f)
}

// Tail returns the trailing string in path after the final slash for a pat ending with a slash.
//
// Examples:
//
//	Tail("/hello/:title/", "/hello/mr/mizerany") == "mizerany"
//	Tail("/:a/", "/x/y/z")                       == "y/z"
//
func Tail(pat, path string) string {
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(pat):
			if pat[len(pat)-1] == '/' {
				return path[i:]
			}
			return ""
		case pat[j] == ':':
			var nextc byte
			_, nextc, j = match(pat, isAlnum, j+1)
			_, _, i = match(path, matchPart(nextc), i)
		case path[i] == pat[j]:
			i++
			j++
		default:
			return ""
		}
	}
	return ""
}

// Labels returns the list of labels defined in pattern.
//
// For example,
//
//   pattern         returns
//   /hello/:title/  [:title]
//   /:a/:b          [:a :b]
func Labels(pattern string) []string {
	var a []string
	for j := 0; j < len(pattern); {
		if pattern[j] == ':' {
			var name string
			name, _, j = match(pattern, isAlnum, j+1)
			a = append(a, ":"+name)
		} else {
			j++
		}
	}
	return a
}

// NotFound replies to the request with an HTTP 404 not found error. It is
// nearly identical to http.NotFound, but returns JSON instead.
func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"message": "Resource not found"}`, http.StatusNotFound)
}

type patHandler struct {
	pat string
	h   chainhttp.Handler
}

func (ph *patHandler) try(path string) (url.Values, bool) {
	p := make(url.Values)
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(ph.pat):
			if len(ph.pat) > 0 && ph.pat[len(ph.pat)-1] == '/' {
				return p, true
			}
			return nil, false
		case ph.pat[j] == ':':
			var name, val string
			var nextc byte
			name, nextc, j = match(ph.pat, isAlnum, j+1)
			val, _, i = match(path, matchPart(nextc), i)
			p.Add(":"+name, val)
		case path[i] == ph.pat[j]:
			i++
			j++
		default:
			return nil, false
		}
	}
	if j != len(ph.pat) {
		return nil, false
	}
	return p, true
}

func matchPart(b byte) func(byte) bool {
	return func(c byte) bool {
		return c != b && c != '/'
	}
}

func match(s string, f func(byte) bool, i int) (matched string, next byte, j int) {
	j = i
	for j < len(s) && f(s[j]) {
		j++
	}
	if j < len(s) {
		next = s[j]
	}
	return s[i:j], next, j
}

func isAlpha(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlnum(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}
