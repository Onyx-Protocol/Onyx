const Connection = require('./connection')
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
   * @param {String} baseUrl - Chain Core URL.
   * @param {String} token - Chain Core client token for API access.
   * @returns {Client}
   */
  constructor(baseUrl, token) {
    this.connection = new Connection(baseUrl, token)

    /**
     * API actions for access tokens
     * @type {module:accessTokensAPI}
     */
    this.accessTokens = accessTokensAPI(this)

    /**
     * API actions for accounts
     * @type {module:accountsAPI}
     */
    this.accounts = accountsAPI(this)

    /**
     * API actions for assets.
     * @type {module:assetsAPI}
     */
    this.assets = assetsAPI(this)

    /**
     * API actions for balances.
     * @type {module:balancesAPI}
     */
    this.balances = balancesAPI(this)

    /**
     * API actions for config.
     * @type {module:configAPI}
     */
    this.config = configAPI(this)

    /**
     * @property {module:mockHsmKeysAPI} keys API actions for Mock HSM keys.
     * @property {Connection} signerConnection Mock HSM signer connection.
     */
    this.mockHsm = {
      keys: mockHsmKeysAPI(this),
      signerConnection: new Connection('http://localhost:1999/mockhsm')
    }

    /**
     * API actions for transactions.
     * @type {module:transactionsAPI}
     */
    this.transactions = transactionsAPI(this)

    /**
     * API actions for transaction feeds.
     * @type {module:transactionFeedsAPI}
     */
    this.transactionFeeds = transactionFeedsAPI(this)

    /**
     * API actions for unspent outputs.
     * @type {module:unspentOutputsAPI}
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
