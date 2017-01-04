const uuid = require('uuid')
const MAX_BLOCK_HEIGHT = (2 * 63) - 1

class TransactionFeed {

  constructor(feed, client) {
    let nextAfter
    let after = feed['after']
    const filter = feed['filter']
    const id = feed['id']
    const alias = feed['alias']

    const ack = () => client.request('/update-transaction-feed', {
      id,
      after: nextAfter,
      previous_after: after
    }).then(() => { after = nextAfter })

    const query = params => client.transactions.query(params)

    /**
     *
     */
    this.consume = (consumer, timeout = 24*60*60) => {
      var self = this

      return new Promise((resolve, reject) => {
        let queryArgs = {
          filter,
          after,
          timeout: (timeout * 1000),
          ascending_with_long_poll: true,
        }

        const nextPage = () => {
          query(queryArgs).then(page => {
            let index = 0
            let prevItem

            const done = shouldAck => {
              let p
              if (shouldAck) {
                p = ack(prevItem)
              } else {
                p = Promise.resolve()
              }
              p.then(resolve).catch(reject)
            }

            const next = shouldAck => {
              let p
              if (shouldAck && prevItem) {
                p = ack(prevItem)
              } else {
                p = Promise.resolve()
              }

              p.then(() => {
                if (index >= page.items.length) {
                  queryArgs = page.next
                  nextPage()
                  return
                }

                prevItem = page.items[index]
                nextAfter = `${prevItem.block_height}:${prevItem.position}-${MAX_BLOCK_HEIGHT}`
                index++

                // Pass the next item to the consumer, as well as three loop
                // operations:
                //
                // - next(shouldAck): maybe ack, then continue/long-poll to next item
                // - done(shouldAck): maybe ack, then terminate the loop by fulfilling the outer promise
                // - fail(err): terminate the loop by rejecting the outer promise.
                //              Use this if you want to bubble an async error up to
                //              the outer promise catch function.
                //
                // The consumer can also terminate the loop by returning a promise
                // that will reject.

                let res = consumer(prevItem, next, done, reject)
                if (res && typeof res.catch === 'function') {
                  res.catch(reject)
                }
              }).catch(reject) // fail consume loop on ack failure, or on thrown exceptions from "then" function
            }

            next()
          }).catch(reject) // fail consume loop on query failure
        }

        nextPage()
      })
    }
  }
}

/**
 * @class
 */
class TransactionFeeds {

  /**
   * constructor - return TransactionFeeds object configured for specified Chain Core.
   *
   * @param {Client} client Configured Chain client object.
   */
  constructor(client) {
    /**
     * Create a new transaction feed.
     *
     * @param {Object} params Parameters for creating Tansaction Feeds.
     * @param {string} params.alias A unique alias for the transaction feed.
     * @param {string} params.filter A valid filter string for the `/list-transactions`
     *                               endpoint. The transaction feed will be composed of future
     *                               transactions that match the filter.
     * @returns {TransactionFeed}
     */
    this.create = (params = {}) => {
      let body = Object.assign({ client_token: uuid.v4() }, params)
      return client.request('/create-transaction-feed', body)
        .then(data => new TransactionFeed(data, client))
    }

    /**
     * Get single transaction feed given an id/alias.
     *
     * @param {Object} params Parameters to get single Tansaction Feed
     * @param {string} params.id The unique ID of a transaction feed. Either `id` or
     *                           `alias` is required.
     * @param {string} params.alias The unique alias of a transaction feed. Either `id` or
     *                              `alias` is required.
     * @returns {TransactionFeed}
     */
    this.get = (params) => {
      return client.request('/get-transaction-feed', params)
      .then(data => new TransactionFeed(data, client))
    }

    /**
     * Delete a transaction feed given an id/alias.
     *
     * @param {Object} params Parameters to delete single Tansaction Feed
     * @param {string} params.id The unique ID of a transaction feed. Either `id` or
     *                           `alias` is required.
     * @param {string} params.alias The unique alias of a transaction feed. Either `id` or
     *                              `alias` is required.
     */
    this.delete = (params) => {
      client.request('/delete-transaction-feed', params)
      .then(data => data)
    }

    /**
     * Returns a page of transaction feeds defined on the core.
     */
    this.query = (params) => {
      return client.request('/list-transaction-feeds', params)
      .then(data => data)
    }
  }
}

module.exports = TransactionFeeds
