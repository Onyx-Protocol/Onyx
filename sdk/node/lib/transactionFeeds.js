const uuid = require('uuid')

/**
 * TransactionFeed
 * @class
 */
class TransactionFeeds {

  /**
   * constructor - return TransactionFeeds object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {
    /**
     * Create a new transaction feed
     */
    this.create = (params = {}) => {
      let body = Object.assign({ client_token: uuid.v4() }, params)
      return client.request('/create-transaction-feed', body)
        .then(data => data)
    },

    /**
     * get feed given an id/alias
     */
    this.get = (params) => {
      return client.request('/get-transaction-feed', params)
      .then(data => data)
    },

    /**
     * delete a transaction feed given an id/alias
     */
    this.delete = (params) => {
      client.request('/delete-transaction-feed', params)
      .then(data => data)
    },

    /**
     *
     */
    this.query = (params) => {
      return client.request('/list-transaction-feed', params)
      .then(data => data)
    }
  }
}

module.exports = TransactionFeeds
