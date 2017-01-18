const shared = require('./shared')

/**
 * There are two APIs in Chain Core: the client API and the network API. Each
 * API is authenticated using access tokens with HTTP Basic Authentication.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/learn-more/authentication}
 * @module accessTokensAPI 
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
     * @returns {Promise<Object>} Newly created access token.
     */
    create: (params, cb) =>
      shared.create(client, '/create-access-token', params, {skipArray: true, cb}),

    /**
     * Get a list of access tokens sorted by descending creation time,
     * optionally filtered by type.
     *
     * Note: maximum list size is 1000 items
     *
     * @param {Query} params - Pagination information.
     * @param {String} [params.type] - Type of access tokens to return.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results.
     */
    query: (params, cb) => {
      params = params || {}
      params.pageSize = 1000
      return shared.query(client, this, '/list-access-tokens', params, {cb})
    },

    /**
     * Delete the specified access token.
     *
     * @param {String} id - Access token ID.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Status of deleted object
     */
    delete: (id, cb) => shared.tryCallback(
      client.request('/delete-access-token', {id: id}),
      cb
    ),
  }
}

module.exports = accessTokens
