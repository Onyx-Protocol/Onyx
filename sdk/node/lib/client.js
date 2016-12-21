const Connection = require('./connection')

const AccessTokens = require('./accessTokens')
const Accounts = require('./accounts')
const Assets = require('./assets')
const Balances = require('./balances')
const MockHsmKeys = require('./mockHsmKeys')
const transactions = require('./transactions')
const UnspentOutputs = require('./unspentOutputs')
const TransactionFeeds = require('./transactionFeeds')


/**
 * Chain API Client
 */
class Client {
  /**
   * constructor - create a new Chain client object capable of interacting with
   * the specified Chain Core
   *
   * @param  {string} baseUrl Chain Core URL
   * @param  {string} token   Chain Core client token for API access
   * @return {Client}
   */
  constructor(baseUrl, token) {
    this.connection = new Connection(baseUrl, token)

    /**
     * API actions for access tokens
     * @type {AccessTokens}
     */
    this.accessTokens = new AccessTokens(this)

    /**
     * API actions for accounts
     * @type {Accounts}
     */
    this.accounts = new Accounts(this)

    /**
     * API actions for assets.
     * @type {Assets}
     */
    this.assets = new Assets(this)

    /**
     * API actions for balances.
     * @type {Balances}
     */
    this.balances = new Balances(this)

    /**
     * @property {MockHsmKeys} keys API actions for Mock HSM keys
     * @property {Connection} signerConnection Mock HSM signer connection.
     */
    this.mockHsm = {
      keys: new MockHsmKeys(this),
      signerConnection: () => new Connection('http://localhost:1999/mockhsm')
    }

    /**
     * API actions for transactions.
     * @type {Transactions}
     */
    this.transactions = transactions(this)

    /**
     * API actions for transaction feeds.
     * @type {TransactionFeeds}
     */
    this.transactionFeeds = new TransactionFeeds(this)

    /**
     * API actions for unspent outputs.
     * @type {UnspentOutputs}
     */
    this.unspentOutputs = new UnspentOutputs(this)
  }


  /**
   * Submit a request to the stored Chain Core connection
   *
   * @param  {string} path
   * @param  {object} [body={}]
   * @return {Promise}
   */
  request(path, body = {}) {
    return this.connection.request(path, body)
  }
}

module.exports = Client
