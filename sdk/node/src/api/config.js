const shared = require('../shared')

/**
 * @typedef {Object} CoreInfo
 *
 * @property {Object} snapshot
 * @property {Number} snapshot.attempt
 * @property {Number} snapshot.height
 * @property {Number} snapshot.size
 * @property {Number} snapshot.downloaded
 * @property {Boolean} snapshot.inProgress

 * @property {Boolean} isConfigured
 * @property {String} configuredAt RFC3339 timestamp
 * @property {Boolean} isSigner
 * @property {Boolean} isGenerator
 * @property {String} generatorUrl
 * @property {String} generatorAccessToken
 * @property {String} blockchainId
 * @property {Number} blockHeight
 * @property {Number} generatorBlockHeight
 * @property {String} generatorBlockHeightFetchedAt RFC3339 timestamp
 * @property {Boolean} isProduction
 * @property {Number} networkRpcVersion
 * @property {String} coreId
 * @property {String} version
 * @property {String} buildCommit
 *
 * @property {String} buildDate
 * Date when the core binary was compiled.
 *
 * The API is not guaranteed to return this field as an RFC3399 timestamp.
 *
 * @property {Object} health
 */

/**
 * Chain Core can be configured as a new blockchain network, or as a node in an
 * existing blockchain network.
 * <br/><br/>
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
     * @param {String} opts.generatorAccessToken - A network access token provided by administrators
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
