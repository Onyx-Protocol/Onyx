const shared = require('../shared')

/**
 * Chain Core can be configured as a new blockchain network, or as a node in an
 * existing blockchain network.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/get-started/configure}
 * @module configAPI
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
     * @returns {Promise<Object>} Status of reset request.
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
     * @returns {Promise<Object>} Status of configuration request.
     */
    configure: (opts = {}, cb) => shared.tryCallback(
      client.request('/configure', opts),
      cb
    ),

    /**
     * Get info on specified Chain Core.
     *
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Requested info of specified Chain Core.
     */
    info: (cb) => shared.tryCallback(
      client.request('/info'),
      cb
    ),
  }
}

module.exports = configAPI
