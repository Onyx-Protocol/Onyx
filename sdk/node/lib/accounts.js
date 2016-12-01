const shared = require('./shared')

/**
 * Account
 * @class
 */
class Accounts {

  /**
   * constructor - return Accounts object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {
    /**
     * Create a new account
     */
    this.create = (params) => shared.create(client, '/create-account', params),

    /**
     * Create multiple new acconts
     */
    this.createBatch = (params) => shared.createBatch(client, '/create-account', params)

    /**
     * Get a list of accounts matching the specified filter
     */
    this.query = (params) => shared.query(client, '/list-accounts', params),

    /**
     * Create a new control program
     */
    this.createControlProgram = (opts = {}) => {
      const body = {type: 'account'}

      if (opts.alias) body.params = { account_alias: opts.alias }
      if (opts.id)    body.params = { account_id: opts.id }

      return shared.create(client, '/create-control-program', body)
    }
  }
}

module.exports = Accounts
