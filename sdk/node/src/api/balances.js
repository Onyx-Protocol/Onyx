const shared = require('../shared')

/**
 * Any balance on the blockchain is simply a summation of unspent outputs.
 * Unlike other queries in Chain Core, balance queries do not return Chain Core
 * objects, only simple sums over the amount fields in a specified list of
 * unspent output objects
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/queries}
 * @typedef {Object} Balance
 * @global
 *
 * @property {Number} amount
 * Sum of the unspent outputs.
 *
 * @property {Object} sumBy
 * List of parameters on which to sum unspent outputs.
 */

/**
* API for interacting with {@link Balance balances}.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/queries}
 * @module BalancesApi
 */
const balancesAPI = (client) => {
  return {
    /**
     * Get one page of balances matching the specified query.
     *
     * @param {Query} params={} - Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<Balance>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'balances', '/list-balances', params, {cb}),

    /**
     * Request all balances matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params={} - Filter and pagination information.
     * @param {QueryProcessor<Balance>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'balances', params, processor, cb),
  }
}

module.exports = balancesAPI
