// FIXME: Microsoft Edge has issues returning errors for responses
// with a 401 status. We should add browser detection to only
// use the ponyfill for unsupported browsers.
const { fetch } = require('fetch-ponyfill')()

const AccessTokens = require('./accessTokens')
const Accounts = require('./accounts')
const Assets = require('./assets')
const balances = require('./balances')
const mockHsm = require('./mockHsm')
const transactions = require('./transactions')
const unspentOutputs = require('./unspentOutputs')
const transactionFeeds = require('./transactionFeeds')

const errors = require('./errors')


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
    this.baseUrl = baseUrl || 'http://localhost:1999'
    this.token = token || ''

    /**
     * API actions for access tokens
     * @type {AccessTokens}
     */
    this.AccessTokens = new AccessTokens(this)

    /**
     * API actions for accounts
     * @type {Accounts}
     */
    this.accounts = new Accounts(this)

    /**
     * API actions for assets
     * @type {Assets}
     */
    this.assets = new Assets(this)

    /**
     * API actions for balances
     * @type {Balances}
     */
    this.balances = balances(this)

    /**
     * API actions for the Mock HSM
     * @type {MockHSM}
     */
    this.mockHsm = mockHsm(this)

    /**
     * API actions for transactions
     * @type {Transactions}
     */
    this.transactions = transactions(this)

    /**
     * API actions for transaction feeds
     * @type {TransactionFeeds}
     */
    this.transactionFeeds = new transactionFeeds(this)

    /**
     * API actions for unspent outputs
     * @type {UnspentOutputs}
     */
    this.unspentOutputs = unspentOutputs(this)
  }

  request(path, body = {}) {
    if (!body) {
      body = {}
    }

    let req = {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',

        // TODO(jeffomatic): The Fetch API has inconsistent behavior between
        // browser implementations and polyfills.
        //
        // - For Edge: we can't use the browser's fetch API because it doesn't
        // always returns a WWW-Authenticate challenge to 401s.
        // - For Safari/Chrome: using fetch-ponyfill (the polyfill) causes
        // console warnings if the user agent string is provided.
        //
        // For now, let's not send the UA string.
        //'User-Agent': 'chain-sdk-js/0.0'
      },
      body: JSON.stringify(body)
    }

    if (this.token) {
      req.headers['Authorization'] = `Basic ${btoa(this.token)}`
    }

    return fetch(this.baseUrl + path, req).catch((err) => {
      throw errors.create(
        errors.types.FETCH,
        'Fetch error: ' + err.toString(),
        {sourceError: err}
      )
    }).then((resp) => {
      let requestId = resp.headers.get('Chain-Request-Id')
      if (!requestId) {
        throw errors.create(
          errors.types.NO_REQUEST_ID,
          'Chain-Request-Id header is missing. There may be an issue with your proxy or network configuration.',
          {response: resp}
        )
      }

      if (resp.status == 204) {
        return {status: 204}
      }

      return resp.json().catch(() => {
        throw errors.create(
          errors.types.JSON,
          'Could not parse JSON response',
          {response: resp, status: resp.status}
        )
      }).then((body) => {
        if (resp.status / 100 == 2) {
          return body
        }

        // Everything else is a status error.
        let errType = null
        if (resp.status == 401) {
          errType = errors.types.UNAUTHORIZED
        } else if (resp.status == 404) {
          errType = errors.types.NOT_FOUND
        } else if (resp.status / 100 == 4) {
          errType = errors.types.BAD_REQUEST
        } else {
          errType = errors.types.SERVER
        }

        throw errors.create(
          errType,
          errors.formatErrMsg(body, requestId),
          {
            response: resp,
            status: resp.status,
            body: body,
            requestId: requestId
          }
        )
      })
    })
  }
}

module.exports = Client
