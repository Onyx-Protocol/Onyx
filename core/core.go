package core

import (
	"context"
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
	errDisabled          = errors.New("this functionality is disabled")
	errBadBlockPub       = errors.New("supplied block pub key is invalid")
	errNoClientTokens    = errors.New("cannot enable client auth without client access tokens")
)

const (
	networkRPCVersion = 3
)

func (a *API) reset(ctx context.Context, req struct {
	Everything bool `json:"everything"`
}) error {
	dataToReset := "blockchain"
	if req.Everything {
		dataToReset = "everything"
	}

	closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf(dataToReset)
	panic("unreached")
}

func (a *API) info(ctx context.Context) (map[string]interface{}, error) {
	if a.config == nil {
		// never configured
		return map[string]interface{}{
			"is_configured": false,
			"version":       config.Version,
			"build_commit":  config.BuildCommit,
			"build_date":    config.BuildDate,
			"build_config":  config.BuildConfig,
		}, nil
	}
	// If we're not the leader, forward to the leader.
	if a.leader.State() == leader.Following {
		var resp map[string]interface{}
		err := a.forwardToLeader(ctx, "/info", nil, &resp)
		return resp, err
	}
	return a.leaderInfo(ctx)
}

func (a *API) leaderInfo(ctx context.Context) (map[string]interface{}, error) {
	var generatorHeight uint64
	var generatorFetched time.Time

	a.downloadingSnapshotMu.Lock()
	snapshot := a.downloadingSnapshot
	a.downloadingSnapshotMu.Unlock()

	localHeight := a.chain.Height()

	if a.config.IsGenerator {
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

	m := map[string]interface{}{
		"state":                             a.leader.State().String(),
		"is_configured":                     true,
		"configured_at":                     a.config.ConfiguredAt,
		"is_signer":                         a.config.IsSigner,
		"is_generator":                      a.config.IsGenerator,
		"generator_url":                     a.config.GeneratorURL,
		"generator_access_token":            obfuscateTokenSecret(a.config.GeneratorAccessToken),
		"blockchain_id":                     a.config.BlockchainID,
		"block_height":                      localHeight,
		"generator_block_height":            generatorHeight,
		"generator_block_height_fetched_at": generatorFetched,
		"network_rpc_version":               networkRPCVersion,
		"core_id":                           a.config.ID,
		"version":                           config.Version,
		"build_commit":                      config.BuildCommit,
		"build_date":                        config.BuildDate,
		"build_config":                      config.BuildConfig,
		"health":                            a.health(),
	}

	// Add in snapshot information if we're downloading a snapshot.
	if snapshot != nil {
		downloadedBytes, totalBytes := snapshot.Progress()
		m["snapshot"] = map[string]interface{}{
			"attempt":     snapshot.Attempt(),
			"height":      snapshot.Height(),
			"size":        totalBytes,
			"downloaded":  downloadedBytes,
			"in_progress": true,
		}
	}
	return m, nil
}

func (a *API) configure(ctx context.Context, x *config.Config) error {
	if a.config != nil {
		return errAlreadyConfigured
	}

	if x.IsGenerator && x.MaxIssuanceWindow.Duration == 0 {
		x.MaxIssuanceWindow.Duration = 24 * time.Hour
	}

	err := config.Configure(ctx, a.db, x)
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
		log.Printf(req.Context(), "no hijacker")
		return
	}
	conn, buf, err := hijacker.Hijack()
	if err != nil {
		log.Printf(req.Context(), "could not hijack connection: %s\n", err)
		return
	}
	err = buf.Flush()
	if err != nil {
		log.Printf(req.Context(), "could not flush connection buffer: %s\n", err)
	}
	err = conn.Close()
	if err != nil {
		log.Printf(req.Context(), "could not close connection: %s\n", err)
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
