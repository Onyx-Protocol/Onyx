package core

import (
	"context"
	"encoding/json"
	"expvar"
	"net/http"
	"strings"
	"time"

	"chain/core/config"
	"chain/core/fetch"
	"chain/core/leader"
	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
)

var (
	errAlreadyConfigured = errors.New("core is already configured; must reset first")
	errUnconfigured      = errors.New("core is not configured")
	errBadBlockPub       = errors.New("supplied block pub key is invalid")
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

func (h *Handler) reset(ctx context.Context, req struct {
	Everything bool `json:"everything"`
}) error {
	if isProduction() {
		return errors.Wrap(errProdReset)
	}

	dataToReset := "blockchain"
	if req.Everything {
		dataToReset = "everything"
	}

	closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf(dataToReset)
	panic("unreached")
}

func (h *Handler) info(ctx context.Context) (map[string]interface{}, error) {
	if h.Config == nil {
		// never configured
		return map[string]interface{}{
			"is_configured": false,
		}, nil
	}
	if leader.IsLeading() {
		return h.leaderInfo(ctx)
	} else {
		var resp map[string]interface{}
		err := h.forwardToLeader(ctx, "/info", nil, &resp)
		return resp, err
	}
}

func (h *Handler) leaderInfo(ctx context.Context) (map[string]interface{}, error) {
	var (
		generatorHeight  *uint64
		generatorFetched *time.Time
		snapshot         = fetch.SnapshotProgress()
		localHeight      = h.Chain.Height()
	)
	if h.Config.IsGenerator {
		now := time.Now()
		generatorHeight = &localHeight
		generatorFetched = &now
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
		// The dashboard will handle nulls correctly.
		if !fetchTime.IsZero() {
			generatorHeight, generatorFetched = &fetchHeight, &fetchTime
		}
	}

	version := json.RawMessage(expvar.Get("version").String())
	buildCommit := json.RawMessage(expvar.Get("buildcommit").String())
	buildDate := json.RawMessage(expvar.Get("builddate").String())

	m := map[string]interface{}{
		"is_configured":                     true,
		"configured_at":                     h.Config.ConfiguredAt,
		"is_signer":                         h.Config.IsSigner,
		"is_generator":                      h.Config.IsGenerator,
		"generator_url":                     h.Config.GeneratorURL,
		"generator_access_token":            obfuscateTokenSecret(h.Config.GeneratorAccessToken),
		"blockchain_id":                     h.Config.BlockchainID,
		"block_height":                      localHeight,
		"generator_block_height":            generatorHeight,
		"generator_block_height_fetched_at": generatorFetched,
		"is_production":                     isProduction(),
		"network_rpc_version":               networkRPCVersion,
		"core_id":                           h.Config.ID,
		"version":                           &version,
		"build_commit":                      &buildCommit,
		"build_date":                        &buildDate,
		"health":                            h.health(),
	}

	// Add in snapshot information if we're downloading a snapshot.
	if snapshot != nil {
		m["snapshot"] = map[string]interface{}{
			"attempt":     snapshot.Attempt,
			"height":      snapshot.Height,
			"size":        snapshot.Size,
			"downloaded":  snapshot.BytesRead(),
			"in_progress": snapshot.InProgress(),
		}
	}
	return m, nil
}

func (h *Handler) configure(ctx context.Context, x *config.Config) error {
	if h.Config != nil {
		return errAlreadyConfigured
	}

	if x.IsGenerator && x.MaxIssuanceWindow == 0 {
		x.MaxIssuanceWindow = 24 * time.Hour
	}

	err := config.Configure(ctx, h.DB, x)
	if err != nil {
		return err
	}

	closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
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
