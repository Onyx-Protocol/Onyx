const shared = require('./shared')

/**
 * @class
 */
class AccessTokens {
  /**
   * constructor - return AccessTokens object configured for specified Chain Core.
   *
   * @param  {Client} client Configured Chain client object.
   */
  constructor(client) {

    /**
     * Create a new access token.
     *
     * @param {Object} params - Parameters for access token creation.
     * @param {string} params.id - User specified, unique identifier.
     * @param {string} params.type - Either 'client' or 'network'.
     */
    this.create = params => shared.create(client, '/create-access-token', params, {skipArray: true})

    /**
     * Get a list of access tokens sorted by descending creation time,
     * optionally filtered by type.
     *
     * Note: maximum list size is 1000 items
     *
     * @param {Filter} [params={}] - Pagination information.
     * @param {string} [params.type] - Type of access tokens to retrun
     * @returns {Promise<Page>} Requested page of results
     */
    this.query = (params = {}) => {
      params.page_size = 1000
      return shared.query(client, this, '/list-access-tokens', params)
    }

    /**
     * Delete the specified access token.
     *
     * @param {string} id - Access token ID.
     */
    this.delete = id => client.request('/delete-access-token', {id: id})
  }
}

module.exports = AccessTokens
