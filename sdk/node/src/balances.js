const shared = require('./shared')

/**
 * @class
 * Any balance on the blockchain is simply a summation of unspent outputs.
 * Unlike other queries in Chain Core, balance queries do not return Chain Core
 * objects, only simple sums over the amount fields in a specified list of
 * unspent output objects
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/queries}
 */
class Balances {
  /**
   * constructor - return Balances object configured for specified Chain Core.
   *
   * @param {Client} client Configured Chain client object.
   */
  constructor(client) {
    /**
     * Get one page of balances matching the specified query.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results.
     */
    this.query = (params, cb) => shared.query(client, this, '/list-balances', params, {cb})

    /**
     * Request all balances matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)
  }
}

module.exports = Balances
