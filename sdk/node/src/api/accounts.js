const shared = require('../shared')

/**
 * An account is an object in Chain Core that tracks ownership of assets on a
 * blockchain by creating and tracking control programs.
 *
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/accounts}
 * @module accountsAPI
 */
const accountsAPI = (client) => {
  /**
   * @typedef accountsAPI~createRequest
   * @type {Object}
   *
   * @property {String} [alias]
   * User specified, unique identifier.
   *
   * @property {string[]} rootXpubs
   * The list of keys used to create control programs under the account.
   *
   * @property {Number} quorum
   * The number of keys required to sign transactions for the account.
   *
   * @property {Object} [tags]
   * User-specified tag structure for the account.
   */

  return {
    /**
     * Create a new account.
     *
     * @param {Accounts~createRequest} params - Parameters for account creation.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Newly created account.
     */
    create: (params, cb) => shared.create(client, '/create-account', params, {cb}),

    /**
     * Create multiple new accounts.
     *
     * @param {Accounts~createRequest[]} params - Parameters for creation of multiple accounts.
     * @param {batchCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {BatchResponse} Newly created accounts.
     */
    createBatch: (params, cb) => shared.createBatch(client, '/create-account', params, {cb}),

    /**
     * Get one page of accounts matching the specified query.
     *
     * @param {Query} params={} Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results
     */
    query: (params, cb) => shared.query(client, this, '/list-accounts', params, {cb}),

    /**
     * Request all accounts matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params={} - Filter information.
     * @param {QueryProcessor} processor - Processing callback.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor) => shared.queryAll(this, params, processor),

    /**
     * Create a new control program, specifying either an account ID or account
     * alias to indicate the controlling party.
     * <br/><br/>
     * More info: {@link https://chain.com/docs/core/build-applications/control-programs#account-control-programs}
     *
     * @param {Object} params Object containing either alias or ID identifying
     *                      account to create control program for.
     * @param {String} [params.alias] - An account alias. Either this or `id` is required.
     * @param {String} [params.id] - An account ID. Either this or `alias` is required.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Newly created control program.
     */
    createControlProgram: (params, cb) => {
      const body = {type: 'account'}

      if (params.alias) body.params = { accountAlias: params.alias }
      if (params.id)    body.params = { accountId: params.id }

      return shared.tryCallback(
        shared.create(client, '/create-control-program', body),
        cb
      )
    }
  }
}

module.exports = accountsAPI
