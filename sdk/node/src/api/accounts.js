const shared = require('../shared')

/**
 * An account is an object in Chain Core that tracks ownership of assets on a
 * blockchain by creating and tracking control programs.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/accounts}
 * @typedef {Object} Account
 * @global
 *
 * @property {String} id
 * Unique account identifier.
 *
 * @property {String} alias
 * User specified, unique identifier.
 *
 * @property {Key[]} keys
 * The list of keys used to create control programs under the account.
 * Signatures from these keys are required for spending funds held in the account.
 *
 * @property {Number} quorum
 * The number of keys required to sign transactions for the account.
 *
 * @property {Object} tags
 * User-specified tag structure for the account.
 */

/**
 * A receiver is an object that wraps an account control program with additional
 * payment information, such as expiration dates.
 *
 * <br/></br>
 * More info: {@link https://chain.com/docs/core/build-applications/control-programs}
 * @typedef {Object} Receiver
 * @global
 *
 * @property {String} controlProgram
 * The underlying control program that will be used in transactions paying to this receiver.
 *
 * @property {String} expiresAt
 * Timestamp indicating when the receiver will cease to be valid, RFC3339 formatted.
 */

/**
 * API for interacting with {@link Account accounts}.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/accounts}
 * @module AccountsApi
 */
const accountsAPI = (client) => {
  /**
   * @typedef {Object} createRequest
   *
   * @property {String} [alias]
   * User specified, unique identifier.
   *
   * @property {String[]} rootXpubs
   * The list of keys used to create control programs under the account.
   *
   * @property {Number} quorum
   * The number of keys required to sign transactions for the account.
   *
   * @property {Object} [tags]
   * User-specified tag structure for the account.
   */

  /**
   * @typedef {Object} updateTagsRequest
   *
   * @property {String} [id]
   * The account ID. Either the ID or alias must be specified, but not both.
   *
   * @property {String} [alias]
   * The account alias. Either the ID or alias must be specified, but not both.
   *
   * @property {Object} tags
   * A new set of tags, which will replace the existing tags.
   */

  /**
   * @typedef {Object} createReceiverRequest
   *
   * @property {String} [accountAlias]
   * The unique alias of the account. accountAlias or accountId must be
   * provided.
   *
   * @property {String} [accountId]
   * The unique ID of the account. accountAlias or accountId must be
   * provided.
   *
   * @property {String} [expiresAt]
   * An RFC3339 timestamp indicating when the receiver will cease to be valid.
   * Defaults to 30 days in the future.
   */

  return {
    /**
     * Create a new account.
     *
     * @param {module:AccountsApi~createRequest} params - Parameters for account creation.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Account>} Newly created account.
     */
    create: (params, cb) => shared.create(client, '/create-account', params, {cb}),

    /**
     * Create multiple new accounts.
     *
     * @param {module:AccountsApi~createRequest[]} params - Parameters for creation of multiple accounts.
     * @param {batchCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<BatchResponse<Account>>} Newly created accounts.
     */
    createBatch: (params, cb) => shared.createBatch(client, '/create-account', params, {cb}),

    /**
     * Update account tags.
     *
     * @param {module:AccountsApi~updateTagsRequest} params - Parameters for updating account tags.
     * @param {objectCallback} [cb] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Success message.
     */
    updateTags: (params, cb) => shared.singletonBatchRequest(client, '/update-account-tags', params, cb),

    /**
     * Update tags for multiple assets.
     *
     * @param {module:AccountsApi~updateTagsRequest[]} params - Parameters for updating account tags.
     * @param {batchCallback} [cb] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<BatchResponse<Object>>} A batch of success responses and/or errors.
     */
    updateTagsBatch: (params, cb) => shared.batchRequest(client, '/update-account-tags', params, cb),

    /**
     * Get one page of accounts matching the specified query.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<Account>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'accounts', '/list-accounts', params, {cb}),

    /**
     * Request all accounts matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {QueryProcessor<Account>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'accounts', params, processor, cb),

    /**
     * Create a new receiver under the specified account.
     *
     * @param {module:AccountsApi~createReceiverRequest} params - Parameters for receiver creation.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Receiver>} Newly created receiver.
     */
    createReceiver: (params, cb) => shared.create(client, '/create-account-receiver', params, {cb}),

    /**
     * Create multiple receivers under the specified accounts.
     *
     * @param {module:AccountsApi~createReceiverRequest[]} params - Parameters for creation of multiple receivers.
     * @param {batchCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<BatchResponse<Receiver>>} Newly created receivers.
     */
    createReceiverBatch: (params, cb) => shared.createBatch(client, '/create-account-receiver', params, {cb}),
  }
}

module.exports = accountsAPI
