package reqid

import (
	"crypto/rand"
	"encoding/hex"
	"log"

	"golang.org/x/net/context"
)

func New() string {
	// Given n IDs of length b bits, the probability that there will be a collision is bounded by
	// the number of pairs of IDs multiplied by the probability that any pair might collide:
	// p ≤ n(n - 1)/2 * 1/(2^b)
	//
	// We assume an upper bound of 1000 req/sec, which means that in a week there will be
	// n = 1000 * 604800 requests. If l = 10, b = 8*10, then p ≤ 1.512e-7, which is a suitably
	// low probability.
	l := 10
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		log.Println("error making reqID")
	}
	return hex.EncodeToString(b)
}

func AddToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "requestID", New())
}

func FromContext(ctx context.Context) string {
	reqID, ok := (ctx.Value("requestID")).(string)
	if !ok {
		return "unknown_req_id"
	}
	return reqID
}
