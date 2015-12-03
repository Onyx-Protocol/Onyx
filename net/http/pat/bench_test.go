package pat

import (
	"net/http"
	"testing"

	"golang.org/x/net/context"

	chainhttp "chain/net/http"
)

func BenchmarkPatternMatching(b *testing.B) {
	p := New()
	p.Get("/hello/:name", chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {}))
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		r, err := http.NewRequest("GET", "/hello/blake", nil)
		if err != nil {
			panic(err)
		}
		b.StartTimer()
		p.ServeHTTPContext(context.Background(), nil, r)
	}
}
