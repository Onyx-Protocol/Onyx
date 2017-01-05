const shared = require('./shared')

/**
 * @class
 */
class Accounts {
  /**
   * @typedef Accounts~createRequest
   * @type {Object}
   *
   * @property {string} [alias]
   * User specified, unique identifier.
   *
   * @property {string[]} root_xpubs
   * The list of keys used to create control programs under the account.
   *
   * @property {number} quorum
   * The number of keys required to sign transactions for the account.
   *
   * @property {Object} [tags]
   * User-specified tag structure for the account.
   */

  /**
   * constructor - return Accounts object configured for specified Chain Core.
   *
   * @param  {Client} client Configured Chain client object.
   */
  constructor(client) {
    /**
     * Create a new account.
     *
     * @param {Accounts~createRequest} params - Parameters for account creation.
     * @param {createCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @return { Promise<Object> } - Newly created account
     */
    this.create = (params, cb) => shared.create(client, '/create-account', params, {cb})

    /**
     * Create multiple new accounts.
     *
     * @param {Accounts~createRequest[]} params - Parameters for creation of multiple accounts.
     * @param {batchCreateCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @return {BatchResponse}
     */
    this.createBatch = (params, cb) => shared.createBatch(client, '/create-account', params, {cb})

    /**
     * Get one page of accounts matching the specified filter.
     *
     * @param {Filter} [params={}] Filter and pagination information.
     * @param {queryCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results
     */
    this.query = (params, cb) => shared.query(client, this, '/list-accounts', params, {cb})

    /**
     * Request all accounts matching the specified filter, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Filter} params Filter and pagination information.
     * @param {QueryProcessor} processor Processing callback.
     * @return {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error
     */
    this.queryAll = (params, processor) => shared.queryAll(this, params, processor)

    /**
     * Create a new control program.
     *
     * @param {Object} params Object containing either alias or ID identifying
     *                      account to create control program for.
     * @param {string} [params.alias]
     * @param {string} [params.id]
     * @param {function} [callback] - Optional callback. Use instead of Promise return value as desired.
     */
    this.createControlProgram = (params, cb) => {
      const body = {type: 'account'}

      if (params.alias) body.params = { account_alias: params.alias }
      if (params.id)    body.params = { account_id: params.id }

      return shared.create(client, '/create-control-program', body)
        .callback(cb)
    }
  }
}

module.exports = Accounts
