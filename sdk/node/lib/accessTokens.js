const shared = require('./shared')

/**
 * AccessTokens
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
     * @param {Object} params - Parameters for access token creation.
     * @param {string} params.id - User specified, unique identifier.
     * @param {string} params.type - Either 'client' or 'network'.
     */
    this.create = (params) => shared.create(client, '/create-access-token', params)

    /**
     * Get a list of access tokens.
     * @param {Filter} [params={}] - Pagination information.
     */
    this.query = (params = {}) => shared.query(client, '/list-access-token', params)

    /**
     * Delete the specified access token.
     * @param {string} id - Access token ID.
     */
    this.delete = (id) => client.request('/delete-access-token', {id: id})
  }
}
