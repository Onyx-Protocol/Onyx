const Connection = require('./connection')
const accessControlAPI = require('./api/accessControl')
const accessTokensAPI = require('./api/accessTokens')
const accountsAPI = require('./api/accounts')
const assetsAPI = require('./api/assets')
const balancesAPI = require('./api/balances')
const configAPI = require('./api/config')
const mockHsmKeysAPI = require('./api/mockHsmKeys')
const transactionsAPI = require('./api/transactions')
const transactionFeedsAPI = require('./api/transactionFeeds')
const unspentOutputsAPI = require('./api/unspentOutputs')

/**
 * The Chain API Client object is the root object for all API interactions.
 * To interact with Chain Core, a Client object must always be instantiated
 * first.
 * @class
 */
class Client {
  /**
   * constructor - create a new Chain client object capable of interacting with
   * the specified Chain Core.
   *
   * Passing a configuration object is the preferred way of calling this constructor.
   * However, to support code written for 1.1 and older, the constructor supports passing
   * in a string URL and an optional string token as the first and second parameter, respectively.
   *
   * @param {Object} opts - Plain JS object containing configuration options.
   * @param {String} opts.baseUrl - Chain Core URL.
   * @param {String} opts.token - Chain Core client token for API access.
   * @returns {Client}
   */
  constructor(opts = {}) {
    // If the first argument is a string,
    // support the deprecated constructor params.
    if (typeof opts === 'string') {
      opts = {
        baseUrl: arguments[0],
        token: arguments[1] || ''
      }
    }
    opts.baseUrl = opts.baseUrl || 'http://localhost:1999'
    this.connection = new Connection(opts.baseUrl, opts.token)

    /**
     * API actions for access tokens
     * @type {module:AccessTokensApi}
     */
    this.accessTokens = accessTokensAPI(this)

    /**
     * API actions for access control grants
     * @type {module:AccessControlApi}
     */
    this.accessControl = accessControlAPI(this)

    /**
     * API actions for accounts
     * @type {module:AccountsApi}
     */
    this.accounts = accountsAPI(this)

    /**
     * API actions for assets.
     * @type {module:AssetsApi}
     */
    this.assets = assetsAPI(this)

    /**
     * API actions for balances.
     * @type {module:BalancesApi}
     */
    this.balances = balancesAPI(this)

    /**
     * API actions for config.
     * @type {module:ConfigApi}
     */
    this.config = configAPI(this)

    /**
     * @property {module:MockHsmKeysApi} keys API actions for MockHSM keys.
     * @property {Connection} signerConnection MockHSM signer connection.
     */
    this.mockHsm = {
      keys: mockHsmKeysAPI(this),
      signerConnection: new Connection(`${opts.baseUrl}/mockhsm`, opts.token)
    }

    /**
     * API actions for transactions.
     * @type {module:TransactionsApi}
     */
    this.transactions = transactionsAPI(this)

    /**
     * API actions for transaction feeds.
     * @type {module:TransactionFeedsApi}
     */
    this.transactionFeeds = transactionFeedsAPI(this)

    /**
     * API actions for unspent outputs.
     * @type {module:UnspentOutputsApi}
     */
    this.unspentOutputs = unspentOutputsAPI(this)
  }


  /**
   * Submit a request to the stored Chain Core connection.
   *
   * @param {String} path
   * @param {object} [body={}]
   * @returns {Promise}
   */
  request(path, body = {}) {
    return this.connection.request(path, body)
  }
}

module.exports = Client
