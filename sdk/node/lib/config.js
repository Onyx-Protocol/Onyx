/**
 * Config
 * @class
 */
class Config {
  /**
   * constructor - return Config object configured for specified Chain Core.
   *
   * @param  {Client} client Configured Chain client object.
   */
  constructor(client) {
    
    /**
     * Reset specified Chain Core.
     *
     * @param {Boolean} everything - If `true`, all objects including access tokens and
     *                               MockHSM keys will be deleted. If `false`, then access tokens
     *                               and MockHSM keys will be preserved.
     */
    this.reset = (everything = false) => client.request('/reset', {everything: everything})

    /**
     * Configure specified Chain Core.
     *
     * @param {Object} opts - options for configuring Chain Core.
     * @param {Boolean} opts.is_generator - Whether the local core will be a block generator
     *                                      for the blockchain; i.e., you are starting a new blockchain on
     *                                      the local core. `false` if you are connecting to a
     *                                      pre-existing blockchain..
     * @param {string} opts.generator_url - A URL for the block generator. Required if
     *                                      `is_generator` is false.
     * @param {string} opts.generator_access_token - A network access token provided by administrators
     *                                               of the block generator. Required if `is_generator` is false.
     * @param {string} opts.blockchain_id - The unique ID of the generator's blockchain.
     *                                      Required if `is_generator` is false.
     */
    this.configure = (opts = {}) => client.request('/configure', opts)

    /**
     * Get info on specified Chain Core.
     *
     * @returns {Promise} Requested info of specified Chain Core.
     */
    this.info = () => client.request('/info')
  }
}

module.exports = Config
