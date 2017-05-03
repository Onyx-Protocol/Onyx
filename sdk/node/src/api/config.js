const shared = require('../shared')

/**
 * Basic information about the configuration of Chain Core, as well as any
 * errors encountered when updating the local state of the blockchain
 *
 * More info: {@link https://chain.com/docs/core/get-started/configure}
 * @typedef {Object} CoreInfo
 *
 * @property {Object} snapshot
 * @property {Number} snapshot.attempt
 * @property {Number} snapshot.height
 * @property {Number} snapshot.size
 * @property {Number} snapshot.downloaded
 * @property {Boolean} snapshot.inProgress
 *
 * @property {Boolean} isConfigured
 * Whether the core has been configured.
 *
 * @property {String} configuredAt
 * RFC3339 timestamp reflecting when the core was configured.
 *
 * @property {Boolean} isSigner
 * Whether the core is configured as a block signer.
 *
 * @property {Boolean} isGenerator
 * Whether the core is configured as the blockchain generator.
 *
 * @property {String} generatorUrl
 * URL of the generator.
 *
 * @property {String} generatorAccessToken
 * The access token used to connect to the generator.
 *
 * @property {String} blockchainId
 * Hash of the initial block.
 *
 * @property {Number} blockHeight
 * Height of the blockchain in the local core.
 *
 * @property {Number} generatorBlockHeight
 * Height of the blockchain in the generator
 *
 * @property {String} generatorBlockHeightFetchedAt
 * RFC3339 timestamp reflecting the last time generator_block_height was updated.
 *
 * @property {Boolean} isProduction
 * Whether the core is running in production mode.
 *
 * @property {Number} crosscoreRpcVersion
 * The cross-core API version supported by this core.
 *
 * @property {Number} networkRpcVersion
 * DEPRECATED. Do not use in 1.2 or greater. Superseded by {@link crosscoreRpcVersion}.
 *
 * @property {String} coreId
 * A random identifier for the core, generated during configuration.
 *
 * @property {String} version
 * The release version of the cored binary.
 *
 * @property {String} buildCommit
 * Git SHA of build source.
 *
 * @property {String} buildDate
 * Unixtime (as string) of binary build.
 *
 * @property {Object} buildConfig
 * Features enabled or disabled in this build of Chain Core.
 *
 * @property {Boolean} buildConfig.isLocalhostAuth
 * Whether any request from the loopback device (localhost) should be
 * automatically authenticated and authorized, without additional
 * credentials.
 *
 * @property {Boolean} buildConfig.isMockHsm
 * Whether the MockHSM API is enabled.
 *
 * @property {Boolean} buildConfig.isReset
 * Whether the core reset API call is enabled.
 *
 * @property {Boolean} buildConfig.isPlainHttp
 * Whether non-TLS HTTP requests (http://...) are allowed.
 *
 * @property {Object} health
 * Blockchain error information.
 */

/**
 * Chain Core can be configured as a new blockchain network, or as a node in an
 * existing blockchain network.
 *
 * More info: {@link https://chain.com/docs/core/get-started/configure}
 * @module ConfigApi
 */
const configAPI = (client) => {
  return {
    /**
     * Reset specified Chain Core.
     *
     * @param {Boolean} everything - If `true`, all objects including access tokens and
     *                               MockHSM keys will be deleted. If `false`, then access tokens
     *                               and MockHSM keys will be preserved.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} Promise resolved on success.
     */
    reset: (everything = false, cb) => shared.tryCallback(
      client.request('/reset', {everything: everything}),
      cb
    ),

    /**
     * Configure specified Chain Core.
     *
     * @param {Object} opts - options for configuring Chain Core.
     * @param {Boolean} opts.isGenerator - Whether the local core will be a block generator
     *                                      for the blockchain; i.e., you are starting a new blockchain on
     *                                      the local core. `false` if you are connecting to a
     *                                      pre-existing blockchain.
     * @param {String} opts.generatorUrl - A URL for the block generator. Required if
     *                                      `isGenerator` is false.
     * @param {String} opts.generatorAccessToken - An access token provided by administrators
     *                                               of the block generator. Required if `isGenerator` is false.
     * @param {String} opts.blockchainId - The unique ID of the generator's blockchain.
     *                                      Required if `isGenerator` is false.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} Promise resolved on success.
     */
    configure: (opts = {}, cb) => shared.tryCallback(
      client.request('/configure', opts),
      cb
    ),

    /**
     * Get info on specified Chain Core.
     *
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<CoreInfo>} Requested info of specified Chain Core.
     */
    info: (cb) => shared.tryCallback(
      client.request('/info'),
      cb
    ),
  }
}

module.exports = configAPI
