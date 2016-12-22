const shared = require('./shared')

/**
 * @class
 */
class Assets {
  /**
   * @typedef Assets~createRequest
   * @type {Object}
   *
   * @property {string} [alias]
   * User specified, unique identifier.
   *
   * @property {string[]} root_xpubs
   * The list of keys used to create the issuance program for the asset.
   *
   * @property {number} quorum
   * The number of keys required to issue units of the asset.
   *
   * @property {Object} [tags]
   * User-specified, arbitrary/unstructured data local to the asset's originating core.
   *
   * @property {Object} [defintion]
   * User-specified, arbitrary/unstructured data visible across blockchain networks.
   */

  /**
   * constructor - return Assets object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {
    /**
     * Create a new asset.
     *
     * @param {Assets~createRequest} params - Parameters for asset creation
     */
    this.create = (params) => shared.create(client, '/create-asset', params)

    /**
     * Create multiple new assets.
     *
     * @param {Assets~createRequest[]} params - Parameters for creation of multiple assets
     */
    this.createBatch = (params) => shared.createBatch(client, '/create-asset', params)

    /**
     * Get one page of assets matching the specified filter
     *
     * @param {Filter} [params={}] Filter and pagination information
     * @returns {Page} Requested page of results
     */
    this.query = (params) => shared.query(client, this, '/list-assets', params)

    /**
     * Request all assets matching the specified filter, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Filter} params Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     */
    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)
  }
}

module.exports = Assets
