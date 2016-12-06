const shared = require('./shared')

/**
 * Account
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
   * constructor - return Accounts object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {
    /**
     * Create a new account
     * @param {Accounts~createRequest} params - Parameters for account creation
     */
    this.create = (params) => shared.create(client, '/create-account', params),

    /**
     * Create multiple new acconts
     * @param {Accounts~createRequest[]} params - Parameters for creation of multiple accounts
     */
    this.createBatch = (params) => shared.createBatch(client, '/create-account', params)

    /**
     * Get a list of accounts matching the specified filter
     * @param {Filter} [params={}] Filter and pagination information
     */
    this.query = (params = {}) => shared.query(client, '/list-accounts', params),

    /**
     * Create a new control program
     * @param {Object} opts Object containing either alias or ID identifying account to create control program for
     * @param {string} [opts.alias]
     * @param {string} [opts.id]
     */
    this.createControlProgram = (opts) => {
      const body = {type: 'account'}

      if (opts.alias) body.params = { account_alias: opts.alias }
      if (opts.id)    body.params = { account_id: opts.id }

      return shared.create(client, '/create-control-program', body)
    }
  }
}

module.exports = Accounts
