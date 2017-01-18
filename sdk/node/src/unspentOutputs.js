const shared = require('./shared')

/**
 * Each new transaction in the blockchain consumes some unspent outputs and
 * creates others. An output is considered unspent when it has not yet been used
 * as an input to a new transaction. All asset units on a blockchain exist in
 * the unspent output set.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/unspent-outputs}
 * @module unspentOutputsAPI
 */
const unspentOutputsAPI = (client) => {
  return {
    /**
     * Get one page of unspent outputs matching the specified query.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results
     */
    query: (params, cb) => shared.query(client, this, '/list-unspent-outputs', params, {cb}),

    /**
     * Request all unspent outputs matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error
     */
    queryAll: (params, processor) => shared.queryAll(this, params, processor),
  }
}

module.exports = unspentOutputsAPI
