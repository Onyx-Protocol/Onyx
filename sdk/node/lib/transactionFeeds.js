const uuid = require('uuid')

/**
 * TransactionFeedItem
 * @class
 */
class TransactionFeedItem {

  constructor(feed, client) {
    this.filter = feed['filter']
    this.after = feed['after']
    this.id = feed['id']
    this.alias = feed['alias']
    /**
     *
     */
    this.consume = (timeout = 24*60*60) => {
      console.log("hey")
      let query = {
         filter: this.filter,
         after: this.after,
         timeout: (timeout * 1000),
         ascending_with_long_poll: true
      }
        client.request('/list-transactions', query)
        .then((page) => {
          console.log(query)
          query = page['next']
          page['items'].forEach((tx) => {
            console.log(tx)
            next_after = tx
          })
        })
     },

    this.ack = () => {
      client.request(
        'update-transaction-feed',
        {
          id: this.id,
          after: next_after,
          previous_after: this.after
        }
      ).then(() => {
        this.after = next_after
        next_after = null
      })

    }
  }
}

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
      .then(data => new TransactionFeedItem(Object.assign(data), client))
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
