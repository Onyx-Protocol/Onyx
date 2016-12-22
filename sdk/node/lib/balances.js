const shared = require('./shared')

/**
 * @class
 */
class Balances {
  /**
   * constructor - return Balances object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {
    /**
     * Get one page of balances matching the specified filter
     *
     * @param {Filter} [params={}] Filter and pagination information
     * @returns {Page} Requested page of results
     */
    this.query = (params) => shared.query(client, this, '/list-balances', params)

    /**
     * Request all balances matching the specified filter, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Filter} params Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     */
    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)
  }
}

module.exports = Balances
