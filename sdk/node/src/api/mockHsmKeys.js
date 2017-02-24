const uuid = require('uuid')
const shared = require('../shared')

 /**
  * Cryptographic private keys are the primary authorization mechanism on a
  * blockchain. For development environments, Chain Core provides a convenient
  * Mock HSM.
  * <br/><br/>
  * More info: {@link https://chain.com/docs/core/build-applications/keys}
  *
  * @typedef {Object} MockHsmKey
  * @global
  *
  * @property {String} alias
  * User specified, unique identifier of the key.
  *
  * @property {String} xpub
  * Hex-encoded string representation of the key.
  */

/**
 * API for interacting with {@link MockHsmKey Mock HSM keys}.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/keys}
 * @module MockHsmKeysApi
 */

const mockHsmKeysAPI = (client) => {
  return {
    /**
     * Create a new MockHsm key.
     *
     * @param {Object} [params={}] - Parameters for Mock HSM key creation.
     * @param {String} params.alias - User specified, unique identifier.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<MockHsmKey>} Newly created Mock HSM key.
     */
    create: (params, cb) => {
      let body = Object.assign({ clientToken: uuid.v4() }, params)
      return shared.tryCallback(
        client.request('/mockhsm/create-key', body).then(data => data),
        cb
      )
    },

    /**
     * Get one page of MockHsm keys, optionally filtered to specified aliases.
     * <br/><br/>
     * <b>NOTE</b>: The <code>filter</code> parameter of {@link Query} is unavailable for the Mock HSM.
     *
     * @param {Query} params={} - Filter and pagination information.
     * @param {Array.<string>} [params.aliases] - List of requested aliases, max 200.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<MockHsmKey>>} Requested page of results.
     */
    query: (params, cb) => {
      if (Array.isArray(params.aliases) && params.aliases.length > 0) {
        params.pageSize = params.aliases.length
      }

      return shared.query(client, 'mockHsm.keys', '/mockhsm/list-keys', params, {cb})
    },

    /**
     * Request all MockHsm keys matching the specified query, calling the
     * supplied processor callback with each item individually.
     * <br/><br/>
     * <b>NOTE</b>: The <code>filter</code> parameter of {@link Query} is unavailable for the Mock HSM.
     *
     * @param {Query} params={} - Pagination information.
     * @param {QueryProcessor<MockHsmKey>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'mockHsm.keys', params, processor, cb),
  }
}

module.exports = mockHsmKeysAPI
