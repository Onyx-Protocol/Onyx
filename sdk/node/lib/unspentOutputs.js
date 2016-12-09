const shared = require('./shared')

/**
 * UnspentOutputs
 * @class
 */
class UnspentOutputs {
  /**
   * constructor - return UnspentOutputs object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {
    /**
     * Get a list of unspent outputs matching the specified filter
     * @param {Filter} [params={}] Filter and pagination information
     */
    this.query = (params) => shared.query(client, '/list-unspent-outputs', params)
  }
}

module.exports = UnspentOutputs
