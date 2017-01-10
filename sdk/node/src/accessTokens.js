const shared = require('./shared')

/**
 * @class
 */
class AccessTokens {
  /**
   * constructor - return AccessTokens object configured for specified Chain Core.
   *
   * @param {Client} client Configured Chain client object.
   */
  constructor(client) {

    /**
     * Create a new access token.
     *
     * @param {Object} params - Parameters for access token creation.
     * @param {string} params.id - User specified, unique identifier.
     * @param {string} params.type - Either 'client' or 'network'.
     * @param {createCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} - Newly created access token
     */
    this.create = (params, cb) =>
      shared.create(client, '/create-access-token', params, {skipArray: true, cb})

    /**
     * Get a list of access tokens sorted by descending creation time,
     * optionally filtered by type.
     *
     * Note: maximum list size is 1000 items
     *
     * @param {Filter} params - Pagination information.
     * @param {string} [params.type] - Type of access tokens to return.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results
     */
    this.query = (params, cb) => {
      params = params || {}
      params.pageSize = 1000
      return shared.query(client, this, '/list-access-tokens', params, {cb})
    }

    /**
     * Delete the specified access token.
     *
     * @param {string} id - Access token ID.
     * @param {function} [callback]
     */
    this.delete = (id, cb) => shared.tryCallback(
      client.request('/delete-access-token', {id: id}),
      cb
    )
  }
}

module.exports = AccessTokens
