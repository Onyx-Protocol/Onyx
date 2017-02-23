const shared = require('../shared')

/**
 * Each new transaction in the blockchain consumes some unspent outputs and
 * creates others. An output is considered unspent when it has not yet been used
 * as an input to a new transaction. All asset units on a blockchain exist in
 * the unspent output set.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/unspent-outputs}
 * @typedef {Object} UnspentOutput
 * @global
 *
 * @property {String} id
 * @property {String} type
 * @property {String} purpose
 * @property {String} transactionId
 * @property {Number} position
 * @property {String} assetId
 * @property {String} assetAlias
 * @property {Object} assetDefinition
 * @property {Object} assetTags
 * @property {Boolean} assetIsLocal
 * @property {Number} amount
 * @property {String} accountId
 * @property {String} accountAlias
 * @property {Object} accountTags
 * @property {String} controlProgram
 * @property {Object} referenceData
 * @property {Boolean} isLocal
 */

/**
 * API for interacting with {@link UnspentOutput unspent outputs}.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/unspent-outputs}
 * @module UnspentOutputsApi
 */
const unspentOutputsAPI = (client) => {
  return {
    /**
     * Get one page of unspent outputs matching the specified query.
     *
     * @param {Query} params={} - Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<UnspentOutput>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'unspentOutputs', '/list-unspent-outputs', params, {cb}),

    /**
     * Request all unspent outputs matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params - Filter and pagination information.
     * @param {QueryProcessor<UnspentOutput>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'unspentOutputs', params, processor, cb),
  }
}

module.exports = unspentOutputsAPI
