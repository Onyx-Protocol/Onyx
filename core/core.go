package core

import (
	"expvar"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"

	"chain/core/config"
	"chain/core/fetch"
	"chain/core/leader"
	"chain/core/pb"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
)

var (
	errAlreadyConfigured = errors.New("core is already configured; must reset first")
	errUnconfigured      = errors.New("core is not configured")
	errBadIssuanceWindow = errors.New("supplied issuance window is invalid")
	errBadBlockchainID   = errors.New("supplied blockchain ID is invalid")
	errNoClientTokens    = errors.New("cannot enable client auth without client access tokens")
	// errProdReset is returned when reset is called on a
	// production system.
	errProdReset = errors.New("reset called on production system")
)

// reserved mockhsm key alias
const (
	networkRPCVersion = 1
)

func isProduction() bool {
	p := expvar.Get("prod")
	return p != nil && p.String() == `"yes"`
}

func (h *Handler) Reset(ctx context.Context, in *pb.ResetRequest) (*pb.ErrorResponse, error) {
	if isProduction() {
		return nil, errors.Wrap(errProdReset)
	}

	dataToReset := "blockchain"
	if in.Everything {
		dataToReset = "everything"
	}

	// TODO(@erykwalder): replace on GRPC
	// closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf(dataToReset)
	panic("unreached")
}

func (h *Handler) Info(ctx context.Context, in *pb.Empty) (*pb.InfoResponse, error) {
	if h.Config == nil {
		// never configured
		return &pb.InfoResponse{IsConfigured: false}, nil
	}
	if leader.IsLeading() {
		return h.leaderInfo(ctx)
	}

	conn, err := leaderConn(ctx, h.DB, h.Addr)
	if err != nil {
		return nil, err
	}
	defer conn.Conn.Close()

	return pb.NewAppClient(conn.Conn).Info(ctx, nil)
}

func (h *Handler) leaderInfo(ctx context.Context) (*pb.InfoResponse, error) {
	var (
		generatorHeight  uint64
		generatorFetched time.Time
		snapshot         = fetch.SnapshotProgress()
		localHeight      = h.Chain.Height()
	)
	if h.Config.IsGenerator {
		now := time.Now()
		generatorHeight = localHeight
		generatorFetched = now
	} else {
		fetchHeight, fetchTime := fetch.GeneratorHeight()
		// Because everything is asynchronous, it's possible for the localHeight to
		// be higher than our cached generator height. In that case, display the
		// local height as the generator height.
		if localHeight > fetchHeight {
			fetchHeight = localHeight
		}

		// fetchTime might be the zero time if we're having trouble connecting
		// to the remote generator. Only set the height & time if we have it.
		// The dashboard will handle zeros correctly.
		if !fetchTime.IsZero() {
			generatorHeight, generatorFetched = fetchHeight, fetchTime
		}
	}

	m := &pb.InfoResponse{
		IsConfigured:                  true,
		ConfiguredAt:                  h.Config.ConfiguredAt.String(),
		IsSigner:                      h.Config.IsSigner,
		IsGenerator:                   h.Config.IsGenerator,
		GeneratorUrl:                  h.Config.GeneratorURL,
		GeneratorAccessToken:          obfuscateTokenSecret(h.Config.GeneratorAccessToken),
		BlockchainId:                  h.Config.BlockchainID[:],
		BlockHeight:                   localHeight,
		GeneratorBlockHeight:          generatorHeight,
		GeneratorBlockHeightFetchedAt: generatorFetched.String(),
		IsProduction:                  isProduction(),
		NetworkRpcVersion:             networkRPCVersion,
		CoreId:                        h.Config.ID,
		Version:                       config.Version,
		BuildCommit:                   config.BuildCommit,
		BuildDate:                     config.BuildDate,
	}

	for k, v := range h.health().Errors {
		if v, ok := v.(string); ok {
			m.Health[k] = v
		}
	}

	// Add in snapshot information if we're downloading a snapshot.
	if snapshot != nil {
		m.Snapshot = &pb.InfoResponse_Snapshot{
			Attempt:    int32(snapshot.Attempt),
			Height:     snapshot.Height,
			Size:       snapshot.Size,
			Downloaded: snapshot.BytesRead(),
			InProgress: snapshot.InProgress(),
		}
	}
	return m, nil
}

func (h *Handler) Configure(ctx context.Context, in *pb.ConfigureRequest) (*pb.ErrorResponse, error) {
	var err error

	if h.Config != nil {
		return nil, errAlreadyConfigured
	}

	x := &config.Config{
		IsSigner:             in.IsSigner,
		IsGenerator:          in.IsGenerator,
		GeneratorURL:         in.GeneratorUrl,
		GeneratorAccessToken: in.GeneratorAccessToken,
		BlockPub:             in.BlockPub,
		Quorum:               int(in.Quorum),
	}

	for _, u := range in.BlockSignerUrls {
		x.Signers = append(x.Signers, config.BlockSigner{
			URL:         u.Url,
			AccessToken: u.AccessToken,
			Pubkey:      u.Pubkey,
		})
	}

	if in.IsGenerator && in.MaxIssuanceWindow == "" {
		x.MaxIssuanceWindow = 24 * time.Hour
	} else if in.IsGenerator {
		x.MaxIssuanceWindow, err = time.ParseDuration(in.MaxIssuanceWindow)
		if err != nil {
			return nil, errors.Wrap(errBadIssuanceWindow, err)
		}
	}

	if in.BlockchainId != nil {
		if len(in.BlockchainId) != len(bc.Hash{}) {
			return nil, errBadBlockchainID
		}
		copy(x.BlockchainID[:], in.BlockchainId)
	}

	err = config.Configure(ctx, h.DB, x)
	if err != nil {
		return nil, err
	}

	// TODO(@erykwalder): replace on GRPC
	// closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf("")
	panic("unreached")
}

func closeConnOK(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Connection", "close")
	w.WriteHeader(http.StatusNoContent)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Messagef(req.Context(), "no hijacker")
		return
	}
	conn, buf, err := hijacker.Hijack()
	if err != nil {
		log.Messagef(req.Context(), "could not hijack connection: %s\n", err)
		return
	}
	err = buf.Flush()
	if err != nil {
		log.Messagef(req.Context(), "could not flush connection buffer: %s\n", err)
	}
	err = conn.Close()
	if err != nil {
		log.Messagef(req.Context(), "could not close connection: %s\n", err)
	}
}

func obfuscateTokenSecret(token string) string {
	toks := strings.SplitN(token, ":", 2)
	var res string
	if len(toks) > 0 {
		res += toks[0]
	}
	if len(toks) > 1 {
		res += ":********"
	}
	return res
}
