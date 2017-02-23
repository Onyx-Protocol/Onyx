const shared = require('../shared')

/**
 * There are two APIs in Chain Core: the client API and the network API. Each
 * API is authenticated using access tokens with HTTP Basic Authentication.
 *
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/learn-more/authentication}
 * @typedef {Object} AccessToken
 * @global
 *
 * @property {String} id
 * User specified, unique identifier.
 *
 * @property {String} token
 * Only returned in the response from {@link AccessTokensApi~create}.
 *
 * @property {String} type
 * Either 'client' or 'network'.
 *
 * @property {String} createdAt
 * Timestamp of token creation, RFC3339 formatted.
 */

/**
 * API for interacting with {@link AccessToken access tokens}
 *
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/learn-more/authentication}
 * @module AccessTokensApi
 */
const accessTokens = (client) => {
  return {
    /**
     * Create a new access token.
     *
     * @param {Object} params - Parameters for access token creation.
     * @param {String} params.id - User specified, unique identifier.
     * @param {String} params.type - Either 'client' or 'network'.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<AccessToken>} Newly created access token.
     */
    create: (params, cb) =>
      shared.create(client, '/create-access-token', params, {skipArray: true, cb}),

    /**
     * Get one page of access tokens sorted by descending creation time,
     * optionally filtered by type.
     *
     * @param {Query} params - Pagination information.
     * @param {String} [params.type] - Type of access tokens to return.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<AccessToken>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'accessTokens', '/list-access-tokens', params, {cb}),

    /**
     * Request all access tokens matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params={} - Pagination information.
     * @param {String} [params.type] - Type of access tokens to return.
     * @param {QueryProcessor} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                    rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'accessTokens', params, processor, cb),

    /**
     * Delete the specified access token.
     *
     * @param {String} id - Access token ID.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Status of deleted object.
     */
    delete: (id, cb) => shared.tryCallback(
      client.request('/delete-access-token', {id: id}),
      cb
    ),
  }
}

module.exports = accessTokens
