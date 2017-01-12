const shared = require('./shared')

/**
 * @class
 * An asset is a type of value that can be issued on a blockchain. All units of
 * a given asset are fungible.
 * <br/><br/>
 * Units of an asset can be transacted directly between parties without the
 * involvement of the issuer.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/assets}
 */
class Assets {
  /**
   * @typedef Assets~createRequest
   * @type {Object}
   *
   * @property {string} [alias]
   * User specified, unique identifier.
   *
   * @property {string[]} rootXpubs
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
   * constructor - return Assets object configured for specified Chain Core.
   *
   * @param {Client} client Configured Chain client object.
   */
  constructor(client) {
    /**
     * Create a new asset.
     *
     * @param {Assets~createRequest} params - Parameters for asset creation.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     */
    this.create = (params, cb) => shared.create(client, '/create-asset', params, {cb})

    /**
     * Create multiple new assets.
     *
     * @param {Assets~createRequest[]} params - Parameters for creation of multiple assets.
     * @param {batchCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     */
    this.createBatch = (params, cb) => shared.createBatch(client, '/create-asset', params, {cb})

    /**
     * Get one page of assets matching the specified query.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results
     */
    this.query = (params) => shared.query(client, this, '/list-assets', params)

    /**
     * Request all assets matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error
     */
    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)
  }
}

module.exports = Assets
