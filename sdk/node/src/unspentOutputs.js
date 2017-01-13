const shared = require('./shared')

/**
 * @class
 */
class UnspentOutputs {
  /**
   * constructor - return UnspentOutputs object configured for specified Chain Core.
   *
   * @param {Client} client Configured Chain client object.
   */
  constructor(client) {
    /**
     * Get one page of unspent outputs matching the specified query.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results
     */
    this.query = (params, cb) => shared.query(client, this, '/list-unspent-outputs', params, {cb})

    /**
     * Request all unspent outputs matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error
     */
    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)
  }
}

module.exports = UnspentOutputs
