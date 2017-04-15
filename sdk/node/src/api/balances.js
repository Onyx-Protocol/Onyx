const shared = require('../shared')

/**
 * Any balance on the blockchain is simply a summation of unspent outputs.
 * Unlike other queries in Chain Core, balance queries do not return Chain Core
 * objects, only simple sums over the amount fields in a specified list of
 * unspent output objects
 *
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
 * 
 * More info: {@link https://chain.com/docs/core/build-applications/queries}
 * @module BalancesApi
 */
const balancesAPI = (client) => {
  return {
    /**
     * Get one page of balances matching the specified query.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Array<String>} params.sumBy - List of unspent output attributes to sum by.
     * @param {Integer} params.timestamp - A millisecond Unix timestamp. By using this parameter, you can perform queries that reflect the state of the blockchain at different points in time.
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<Balance>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'balances', '/list-balances', params, {cb}),

    /**
     * Request all balances matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Array<String>} params.sumBy - List of unspent output attributes to sum by.
     * @param {Integer} params.timestamp - A millisecond Unix timestamp. By using this parameter, you can perform queries that reflect the state of the blockchain at different points in time.
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {QueryProcessor<Balance>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'balances', params, processor, cb),
  }
}

module.exports = balancesAPI
