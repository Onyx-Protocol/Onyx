package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"chain/core/config"
	"chain/core/fetch"
	"chain/core/leader"
	"chain/database/sinkdb"
	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
	"chain/net/raft"
	"chain/protocol/bc"
)

var (
	errAlreadyConfigured = errors.New("core is already configured; must reset first")
	errUnconfigured      = errors.New("core is not configured")
	errNoMockHSM         = errors.New("core is not configured with a mockhsm")
	errNoReset           = errors.New("core is not configured with reset capabilities")
	errBadBlockPub       = errors.New("supplied block pub key is invalid")
	errNoClientTokens    = errors.New("cannot enable client auth without client access tokens")
)

const (
	crosscoreRPCVersion = 3
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

	var (
		configuredAtSecs  int64 = int64(a.config.ConfiguredAt / 1000)
		configuredAtNSecs int64 = int64((a.config.ConfiguredAt % 1000) * 1e6)
	)

	m := map[string]interface{}{
		"state":                             a.leader.State().String(),
		"is_configured":                     true,
		"configured_at":                     time.Unix(configuredAtSecs, configuredAtNSecs).UTC(),
		"is_signer":                         a.config.IsSigner,
		"is_generator":                      a.config.IsGenerator,
		"generator_url":                     a.config.GeneratorUrl,
		"generator_access_token":            obfuscateTokenSecret(a.config.GeneratorAccessToken),
		"blockchain_id":                     a.config.BlockchainId,
		"block_height":                      localHeight,
		"generator_block_height":            generatorHeight,
		"generator_block_height_fetched_at": generatorFetched,
		"network_rpc_version":               crosscoreRPCVersion, // "Network" is legacy terminology for "Cross-core"
		"crosscore_rpc_version":             crosscoreRPCVersion,
		"core_id":                           a.config.Id,
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

type configureRequest struct {
	// Config is the old-style monolithic Config object. If any of its
	// fields are present in the request, the Chain Core must not already
	// be configured.
	config.Config

	// Updates contains incremental updates to configuration options.
	Updates []configUpdate `json:"updates"`
}

type configUpdate struct {
	Op    string   `json:"op"`
	Key   string   `json:"key"`
	Tuple []string `json:"tuple,omitempty"`
}

// configure implements the RPC handler for the /configure endpoint.
//
// Chain Core has two types of config settings:
// - the monolithic config.Config struct/protobuf that is required
//   before a Chain Core can participate in any blockchain network.
// - individual options set via the config.Options type. Some Chain
//   Core features may be gated on the presence of options.
//
// Eventually if possible, we'd like to replace the monolithic config
// type with the incremental config options.
func (a *API) configure(ctx context.Context, req configureRequest) error {
	// First, apply any of the incremental config updates as one
	// single, atomic sinkdb batch.
	var ops []sinkdb.Op
	for _, update := range req.Updates {
		switch update.Op {
		case "add":
			ops = append(ops, a.options.Add(update.Key, update.Tuple))
		case "add-or-update":
			ops = append(ops, a.options.AddOrUpdate(update.Key, update.Tuple))
		case "rm":
			ops = append(ops, a.options.Remove(update.Key, update.Tuple))
		default:
			return errors.WithDetailf(config.ErrConfigOp, "Unknown config operation %q.", update.Op)
		}
	}

	// If the old way of configuring a single HSM is used,
	// transparently update the config options.
	if req.Config.BlockHsmUrl != "" {
		tup := []string{req.Config.BlockHsmUrl, req.Config.BlockHsmAccessToken}
		ops = append(ops, a.options.Add("enclaves", tup))
	}

	err := a.sdb.Exec(ctx, ops...)
	if err != nil {
		return err
	}
	// TODO(jackson): make the config.Configure atomic with the above
	// incremental updates.

	// If the monolithic Config is populated, also perform the
	// one-time configure of the Core.
	if proto.Equal(&req.Config, &config.Config{}) {
		return nil
	}
	if a.config != nil {
		return errAlreadyConfigured
	}
	if req.Config.IsGenerator && req.Config.MaxIssuanceWindowMs == 0 {
		req.Config.MaxIssuanceWindowMs = bc.DurationMillis(24 * time.Hour)
	}
	err = config.Configure(ctx, a.db, a.sdb, a.httpClient, &req.Config)
	if err != nil {
		return err
	}

	closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf("")
	panic("unreached")
}

func (a *API) retrieveConfig(ctx context.Context, x struct {
	Keys []string `json:"keys"`
}) (map[string][][]string, error) {
	// TODO(jackson): This waits for len(x.Keys) consensus
	// rounds. We should batch the reads instead.
	results := make(map[string][][]string)
	for _, key := range x.Keys {
		tups, err := a.options.List(ctx, key)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results[key] = tups
	}
	return results, nil
}

func CheckConfigMaybeExec(ctx context.Context, sdb *sinkdb.DB, nodeAddr string) {
	conf, err := config.CheckConfigExists(ctx, sdb)
	if err != nil && errors.Root(err) != raft.ErrUninitialized {
		log.Fatalkv(ctx, log.KeyError, err)
	}
	if conf != nil {
		execSelf("")
	}
}

func (a *API) initCluster(ctx context.Context) error {
	err := a.sdb.RaftService().Init()
	if err != nil {
		return err
	}
	if config.BuildConfig.MockHSM {
		log.Printkv(ctx, "warning", "this core uses a mockhsm. mockhsm data does not sync across coreds")
	}

	// TODO(jackson): make adding this process's address
	// atomic with initializing the cluster

	// add this process's address as an allowed member
	err = a.addAllowedMember(ctx, struct{ Addr string }{a.addr})
	return err
}

func (a *API) joinCluster(ctx context.Context, x struct {
	BootAddress string `json:"boot_address"`
}) error {
	if err := validateAddress(x.BootAddress); err != nil {
		return err
	}

	bootURL := fmt.Sprintf("https://%s", x.BootAddress)
	err := a.sdb.RaftService().Join(bootURL)
	if err != nil {
		return err
	}

	if config.BuildConfig.MockHSM {
		log.Printkv(ctx, "warning", "this core uses a mockhsm. mockhsm data does not sync across coreds")
	}
	// The cluster we joined might already be configured. Exec self
	// to restart cored and attempt to load the config.
	closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf("")
	panic("unreached")
}

func (a *API) evict(ctx context.Context, x struct {
	NodeAddress string `json:"node_address"`
}) error {
	if err := validateAddress(x.NodeAddress); err != nil {
		return err
	}
	return a.sdb.RaftService().Evict(ctx, x.NodeAddress)
}

func validateAddress(addr string) error {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		newerr := errors.Sub(errInvalidAddr, err)
		if addrErr, ok := err.(*net.AddrError); ok {
			newerr = errors.WithDetail(newerr, addrErr.Err)
		}
		return newerr
	}
	return nil
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
