# Operations guide

### Topics

- [Monitoring and health checks](#monitoring-and-health-checks)

## Monitoring and health checks

Chain Core exposes two HTTP endpoints for monitoring.

### `/health`

For uptime monitoring, check `/health` periodically. If your request returns anything but a 200 status code, then the server is unavailable.

This endpoint is **unauthenticated**.

### `/info`

The `/info` endpoint reports basic information about the configuration of Chain Core, as well as any errors encountered when updating the local state of the blockchain. These errors include problems with generating new blocks (if the core is a generator), or problems making requests to the generator core.

This endpoint is **authenticated** via HTTP Basic Auth. Your client API token can be used as a username/password pair, e.g.:

```
GET https://<client API token>@chaincore.example.com/info
```

#### Response

The response is a JSON object with the following fields:

Field | Type | Description
--- | --- | ---
`block_height` | integer | Height of the blockchain in the local core
`blockchain_id` | string | Hash of the initial block
`build_commit` | string | Git SHA of build source
`build_date` | string | Unixtime (as string) of binary build
`configured_at` | string | RFC3339 timestamp reflecting when the core was configured
`core_id` | string | A random identifier for the core, generated during configuration
`generator_access_token` | string | The access token used to connect to the generator
`generator_block_height` | integer | Height of the blockchain in the generator
`generator_block_height_fetched_at` | string | RFC3339 timestamp reflecting the last time `generator_block_height` was updated
`generator_url` | string | URL of the generator
`health` | object | **Blockchain error information (see below)**
`is_configured` | boolean | Whether the core has been configured
`is_generator` | boolean | Whether the core is configured as the blockchain generator
`is_production` | boolean | Whether the core is running in production mode
`is_signer` | boolean | Whether the core is configured as a block signer
`network_rpc_version` | 1 | The network version supported by this core
`version` | `1.0.2` | The release version of the `cored` binary

The `health` object has the following structure:

```
{
  "errors": {
    "fetch": <null or string>,
    "generator": <null or string>
  }
}
```

There are two types of errors:

- **fetch** errors occur when the core encounters errors synchronizing blocks from the generator. Among other things, this could mean the generator is not reachable from this core. This field is undefined if the core is a generator.
- **generator** errors occur when the core is acting as a generator, and encounters errors generating a new block. This field is undefined if the core is not a generator.

These fields will be `null` if no errors have been encountered.


