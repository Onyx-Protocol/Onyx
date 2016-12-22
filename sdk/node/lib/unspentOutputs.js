const shared = require('./shared')

/**
 * @class
 */
class UnspentOutputs {
  /**
   * constructor - return UnspentOutputs object configured for specified Chain Core.
   *
   * @param  {Client} client Configured Chain client object.
   */
  constructor(client) {
    /**
     * Get a list of unspent outputs matching the specified filter
     * @param {Filter} [params={}] Filter and pagination information
     */
    this.query = (params) => shared.query(client, this, '/list-unspent-outputs', params)

    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)
  }
}

module.exports = UnspentOutputs
