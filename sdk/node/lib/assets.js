const shared = require('./shared')

/**
 * Assets
 * @class
 */
class Assets {
  /**
   * constructor - return Assets object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {
    /**
     * Create a new asset
     */
    this.create = (params) => shared.create(client, '/create-asset', params)

    /**
     * Get a list of assets matching the specified filter
     */
    this.query = (params) => shared.query(client, '/list-assets', params)
  }
}

module.exports = Assets
