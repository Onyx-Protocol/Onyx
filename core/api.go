// Package core implements Chain Core and its API.
package core

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	libcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"chain/core/accesstoken"
	"chain/core/account"
	"chain/core/asset"
	"chain/core/config"
	"chain/core/leader"
	"chain/core/mockhsm"
	"chain/core/pb"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/rpc"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/core/txfeed"
	"chain/database/pg"
	"chain/errors"
	"chain/generated/dashboard"
	"chain/generated/docs"
	"chain/net/http/limit"
	"chain/net/http/reqid"
	"chain/net/http/static"
	"chain/protocol"
	"chain/protocol/bc"
)

const (
	defGenericPageSize = 100
)

// TODO(kr): change this to "network" or something.
const networkRPCPrefix = "/rpc/"

var (
	errNotFound       = errors.New("not found")
	errRateLimited    = errors.New("request limit exceeded")
	errLeaderElection = errors.New("no leader; pending election")
)

// Handler serves the Chain HTTP API
type Handler struct {
	Chain         *protocol.Chain
	Store         *txdb.Store
	PinStore      *pin.Store
	Assets        *asset.Registry
	Accounts      *account.Manager
	HSM           *mockhsm.HSM
	Indexer       *query.Indexer
	TxFeeds       *txfeed.Tracker
	AccessTokens  *accesstoken.CredentialStore
	Config        *config.Config
	Submitter     txbuilder.Submitter
	DB            pg.DB
	Addr          string
	AltAuth       func(context.Context) bool
	Signer        func(context.Context, *bc.Block) ([]byte, error)
	RequestLimits []RequestLimit

	once           sync.Once
	handler        http.Handler
	actionDecoders map[string]func(data []byte) (txbuilder.Action, error)

	healthMu     sync.Mutex
	healthErrors map[string]interface{}

	auth *apiAuthn
}

type RequestLimit struct {
	Key       func(context.Context) string
	Burst     int
	PerSecond int
	limiter   *limit.BucketLimiter
}

func (h *Handler) Server(crt *tls.Certificate) *grpc.Server {
	h.auth = &apiAuthn{
		tokens:   h.AccessTokens,
		tokenMap: make(map[string]tokenResult),
		alt:      h.AltAuth,
	}

	for _, lim := range h.RequestLimits {
		lim.limiter = limit.NewBucketLimiter(lim.PerSecond, lim.Burst)
	}

	var opts []grpc.ServerOption

	opts = append(opts, grpc.RPCCompressor(grpc.NewGZIPCompressor()))
	opts = append(opts, grpc.RPCDecompressor(grpc.NewGZIPDecompressor()))
	opts = append(opts, grpc.UnaryInterceptor(h.unaryInterceptor))
	opts = append(opts, grpc.MaxMsgSize(1e6))
	if crt != nil {
		opts = append(opts, grpc.Creds(credentials.NewServerTLSFromCert(crt)))
	}
	srv := grpc.NewServer(opts...)

	pb.RegisterNodeServer(srv, h)
	if h.Config != nil && h.Config.IsSigner {
		pb.RegisterSignerServer(srv, h)
	}
	pb.RegisterHSMServer(srv, h)
	pb.RegisterAppServer(srv, h)

	return srv
}

func webAssetsHandler(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", static.Handler{
		Assets:  dashboard.Files,
		Default: "index.html",
	}))
	mux.Handle("/docs/", http.StripPrefix("/docs/", static.Handler{
		Assets: docs.Files,
		Index:  "index.html",
	}))
	mux.Handle("/", next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			http.Redirect(w, req, "/dashboard/", http.StatusFound)
			return
		}

		mux.ServeHTTP(w, req)
	})
}

func leaderConn(ctx context.Context, db pg.DB, self string) (*rpc.GRPCConn, error) {
	addr, err := leader.Address(ctx, db)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	// Don't infinite loop if the leader's address is our own address.
	// This is possible if we just became the leader. The client should
	// just retry.
	if addr == self {
		return nil, errLeaderElection
	}

	conn, err := rpc.NewGRPCConn(addr, "", "", "")
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func healthHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/health" {
			return
		}
		handler.ServeHTTP(w, req)
	})
}

func (h *Handler) unaryInterceptor(ctx libcontext.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var err error
	ctx = reqid.NewContext(ctx, reqid.New())

	if err := h.limit(ctx); err != nil {
		return nil, err
	}

	ctx, err = h.auth.authRPC(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}

	if md, ok := metadata.FromContext(ctx); ok {
		if len(md[rpc.HeaderTimeout]) == 1 {
			timeout, err := time.ParseDuration(md[rpc.HeaderTimeout][0])
			if err == nil {
				var cancel func()
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
		}
		if len(md[rpc.HeaderCoreID]) == 1 {
			countCore(md[rpc.HeaderCoreID][0])
		}
	}

	if h.Config == nil {
		switch info.FullMethod {
		case "/pb.App/Info":
		case "/pb.App/Configure":
		case "/pb.App/CreateAccessToken":
		case "/pb.App/ListAccessTokens":
		case "/pb.App/DeleteAccessToken":
		default:
			return nil, errUnconfigured
		}
	}

	err = grpc.SetHeader(ctx, metadata.Pairs("BlockchainID", h.Config.BlockchainID.String()))
	if err != nil {
		return nil, err
	}

	if l := latency(info.FullMethod); l != nil {
		defer l.RecordSince(time.Now())
	}
	resp, err := handler(ctx, req)
	if err != nil {
		logHTTPError(ctx, err)
		resp = &pb.ErrorResponse{Error: protobufErr(err)}
	}
	return resp, nil
}

func (r *Handler) limit(ctx context.Context) error {
	for _, lim := range r.RequestLimits {
		if !lim.limiter.Allow(lim.Key(ctx)) {
			return errRateLimited
		}
	}
	return nil
}

func PeerID(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	return p.Addr.String()
}

func AuthID(ctx context.Context) string {
	u, _ := userPwFromContext(ctx)
	return u
}
