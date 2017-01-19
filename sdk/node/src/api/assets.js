const shared = require('../shared')

/**
 * An asset is a type of value that can be issued on a blockchain. All units of
 * a given asset are fungible.
 * <br/><br/>
 * Units of an asset can be transacted directly between parties without the
 * involvement of the issuer.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/assets}
 * @module AssetsApi
 */
const assetsAPI = (client) => {
  /**
   * @typedef Assets~createRequest
   * @type {Object}
   *
   * @property {String} [alias]
   * User specified, unique identifier.
   *
   * @property {string[]} rootXpubs
   * The list of keys used to create the issuance program for the asset.
   *
   * @property {Number} quorum
   * The number of keys required to issue units of the asset.
   *
   * @property {Object} [tags]
   * User-specified, arbitrary/unstructured data local to the asset's originating core.
   *
   * @property {Object} [defintion]
   * User-specified, arbitrary/unstructured data visible across blockchain networks.
   */

  return {
    /**
     * Create a new asset.
     *
     * @param {Assets~createRequest} params - Parameters for asset creation.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     */
    create: (params, cb) => shared.create(client, '/create-asset', params, {cb}),

    /**
     * Create multiple new assets.
     *
     * @param {Assets~createRequest[]} params - Parameters for creation of multiple assets.
     * @param {batchCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     */
    createBatch: (params, cb) => shared.createBatch(client, '/create-asset', params, {cb}),

    /**
     * Get one page of assets matching the specified query.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results
     */
    query: (params, cb) => shared.query(client, 'assets', '/list-assets', params, {cb}),

    /**
     * Request all assets matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error
     */
    queryAll: (params, processor) => shared.queryAll(client, 'assets', params, processor),
  }
}

module.exports = assetsAPI
