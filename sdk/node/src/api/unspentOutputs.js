const shared = require('../shared')

/**
 * Each new transaction in the blockchain consumes some unspent outputs and
 * creates others. An output is considered unspent when it has not yet been used
 * as an input to a new transaction. All asset units on a blockchain exist in
 * the unspent output set.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/unspent-outputs}
 * @typedef {Object} UnspentOutput
 * @global
 *
 * @property {String} id
 * Unique transaction identifier.
 *
 * @property {String} type
 * The type of the output. Possible values are "control" and "retire".
 *
 * @property {String} purpose
 * The purpose of the output. Possible values are "receive" and "change".
 *
 * @property {String} transactionId
 * The transaction containing the output.
 *
 * @property {Number} position
 * The output's position in a transaction's list of outputs.
 *
 * @property {String} assetId
 * The id of the asset being issued or spent.
 *
 * @property {String} assetAlias
 * The alias of the asset being issued or spent (possibly null).
 *
 * @property {Object} assetDefinition
 * The definition of the asset being issued or spent (possibly null).
 *
 * @property {Object} assetTags
 * The tags of the asset being issued or spent (possibly null).
 *
 * @property {Boolean} assetIsLocal
 * A flag indicating whether the asset being issued or spent is local.
 *
 * @property {Number} amount
 * The number of units of the asset being issued or spent.
 *
 * @property {String} accountId
 * The id of the account transferring the asset (possibly null).
 *
 * @property {String} accountAlias
 * The alias of the account transferring the asset (possibly null).
 *
 * @property {Object} accountTags
 * The tags associated with the account (possibly null).
 *
 * @property {String} controlProgram
 * The control program which must be satisfied to transfer this output.
 *
 * @property {Object} referenceData
 * User specified, unstructured data embedded within an input (possibly null).
 *
 * @property {Boolean} isLocal
 * A flag indicating if the input is local.
 */

/**
 * API for interacting with {@link UnspentOutput unspent outputs}.
 * 
 * More info: {@link https://chain.com/docs/core/build-applications/unspent-outputs}
 * @module UnspentOutputsApi
 */
const unspentOutputsAPI = (client) => {
  return {
    /**
     * Get one page of unspent outputs matching the specified query.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Integer} params.timestamp - A millisecond Unix timestamp. By using this parameter, you can perform queries that reflect the state of the blockchain at different points in time.
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<UnspentOutput>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'unspentOutputs', '/list-unspent-outputs', params, {cb}),

    /**
     * Request all unspent outputs matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Integer} params.timestamp - A millisecond Unix timestamp. By using this parameter, you can perform queries that reflect the state of the blockchain at different points in time.
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {QueryProcessor<UnspentOutput>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'unspentOutputs', params, processor, cb),
  }
}

module.exports = unspentOutputsAPI
