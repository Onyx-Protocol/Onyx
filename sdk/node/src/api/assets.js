const shared = require('../shared')

/**
 * An asset is a type of value that can be issued on a blockchain. All units of
 * a given asset are fungible. Units of an asset can be transacted directly
 * between parties without the involvement of the issuer.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/assets}
 * @typedef {Object} Asset
 * @global
 *
 * @property {String} id
 * Globally unique identifier of the asset.
 * Asset version 1 specifies the asset id as the hash of:
 * - the asset version
 * - the asset's issuance program
 * - the core's VM version
 * - the hash of the network's initial block
 *
 * @property {String} alias
 * User specified, unique identifier.
 *
 * @property {String} issuanceProgram
 *
 * @property {Key[]} keys
 * The list of keys used to issue units of the asset.
 *
 * @property {Number} quorum
 * The number of signatures required to issue new units of the asset.
 *
 * @property {Object} defintion
 * User-specified, arbitrary/unstructured data visible across
 * blockchain networks. Version 1 assets specify the definition in their
 * issuance programs, rendering the definition immutable.
 *
 * @property {Object} tags
 * User-specified tag structure for the asset.
 */

/**
 * API for interacting with {@link Asset assets}.
 * 
 * More info: {@link https://chain.com/docs/core/build-applications/assets}
 * @module AssetsApi
 */
const assetsAPI = (client) => {
  /**
   * @typedef {Object} createRequest
   *
   * @property {String} [alias]
   * User specified, unique identifier.
   *
   * @property {String[]} rootXpubs
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

  /**
   * @typedef {Object} updateTagsRequest
   *
   * @property {String} [id]
   * The asset ID. Either the ID or alias must be specified, but not both.
   *
   * @property {String} [alias]
   * The asset alias. Either the ID or alias must be specified, but not both.
   *
   * @property {Object} [tags]
   * A new set of tags, which will replace the existing tags.
   */

  return {
    /**
     * Create a new asset.
     *
     * @param {module:AssetsApi~createRequest} params - Parameters for asset creation.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Asset>} Newly created asset.
     */
    create: (params, cb) => shared.create(client, '/create-asset', params, {cb}),

    /**
     * Create multiple new assets.
     *
     * @param {module:AssetsApi~createRequest[]} params - Parameters for creation of multiple assets.
     * @param {batchCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<BatchResponse<Asset>>} Newly created assets.
     */
    createBatch: (params, cb) => shared.createBatch(client, '/create-asset', params, {cb}),

    /**
     * Update asset tags.
     *
     * @param {module:AssetsApi~updateTagsRequest} params - Parameters for updating asset tags.
     * @param {objectCallback} [cb] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Success message.
     */
    updateTags: (params, cb) => shared.singletonBatchRequest(client, '/update-asset-tags', params, cb),

    /**
     * Update tags for multiple assets.
     *
     * @param {module:AssetsApi~updateTagsRequest[]} params - Parameters for updating asset tags.
     * @param {batchCallback} [cb] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<BatchResponse<Object>>} A batch of success responses and/or errors.
     */
    updateTagsBatch: (params, cb) => shared.batchRequest(client, '/update-asset-tags', params, cb),

    /**
     * Get one page of assets matching the specified query.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<Asset>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'assets', '/list-assets', params, {cb}),

    /**
     * Request all assets matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {QueryProcessor<Asset>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'assets', params, processor, cb),
  }
}

module.exports = assetsAPI
