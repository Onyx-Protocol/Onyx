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
    this.create = params => shared.create(client, '/create-access-token', params, {skipArray: true})

    /**
     * Get a list of access tokens, optionally filtered by type.
     * @param {Filter} [params={}] - Pagination information.
     * @param {string} [params.type] - Type of access tokens to retrun
     */
    this.query = (params = {}) => shared.query(client, this, '/list-access-tokens', params)

    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)

    /**
     * Delete the specified access token.
     * @param {string} id - Access token ID.
     */
    this.delete = id => client.request('/delete-access-token', {id: id})
  }
}

module.exports = AccessTokens
