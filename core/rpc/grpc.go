package rpc

import (
	"errors"
	"strings"
	"time"

	libcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type coreCreds struct {
	Username, Password string
}

type GRPCConn struct {
	Conn         *grpc.ClientConn
	BlockchainID string
	CoreID       string
}

func newRPCCreds(token string) (credentials.PerRPCCredentials, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return nil, errors.New("invalid token string")
	}
	return &coreCreds{parts[0], parts[1]}, nil
}

func (c *coreCreds) GetRequestMetadata(ctx libcontext.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"username": c.Username,
		"password": c.Password,
	}, nil
}
func (c *coreCreds) RequireTransportSecurity() bool { return false }

func NewGRPCConn(addr, accesstoken, coreID, blockchainID string) (*GRPCConn, error) {
	conn := &GRPCConn{}

	var opts []grpc.DialOption
	if accesstoken != "" {
		creds, err := newRPCCreds(accesstoken)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithPerRPCCredentials(creds))
	}
	opts = append(opts, grpc.WithCompressor(grpc.NewGZIPCompressor()))
	opts = append(opts, grpc.WithDecompressor(grpc.NewGZIPDecompressor()))
	if strings.HasPrefix(addr, "localhost") || strings.HasPrefix(addr, "127.0.0.1") || strings.HasPrefix(addr, ":") {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithUnaryInterceptor(conn.interceptor))

	gconn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	conn.Conn = gconn
	return conn, nil
}

func (c *GRPCConn) interceptor(ctx libcontext.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	md := metadata.New(nil)
	opts = append(opts, grpc.Header(&md))

	var ctxPairs []string

	ctxPairs = append(ctxPairs, HeaderCoreID, c.CoreID)

	deadline, ok := ctx.Deadline()
	if ok {
		ctxPairs = append(ctxPairs, HeaderTimeout, deadline.Sub(time.Now()).String())
	}

	ctx = metadata.NewContext(ctx, metadata.Pairs(ctxPairs...))

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		return err
	}

	var blockchainID string
	if len(md["BlockchainID"]) == 1 {
		blockchainID = md["BlockchainID"][0]
	}

	if blockchainID != "" && c.BlockchainID != "" && blockchainID != c.BlockchainID {
		return ErrWrongNetwork
	}

	return nil
}
