const shared = require('../shared')

const uuid = require('uuid')
const MAX_BLOCK_HEIGHT = (2 * 63) - 1

/**
 * @class
 * A single transaction feed that can be consumed. See {@link TransactionFeeds}
 * for actions to create TransactionFeed objects.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/real-time-transaction-processing}
 */
class TransactionFeed {

  constructor(feed, client) {
    let nextAfter
    let after = feed['after']
    const filter = feed['filter']
    const id = feed['id']

    const ack = () => client.request('/update-transaction-feed', {
      id,
      after: nextAfter,
      previousAfter: after
    }).then(() => { after = nextAfter })

    const query = params => client.transactions.query(params)

    /**
     *
     */
    this.consume = (consumer, timeout = 24*60*60) => {
      return new Promise((resolve, reject) => {
        let queryArgs = {
          filter,
          after,
          timeout: (timeout * 1000),
          ascendingWithLongPoll: true,
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
                nextAfter = `${prevItem.blockHeight}:${prevItem.position}-${MAX_BLOCK_HEIGHT}`
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
 * You can use transaction feeds to process transactions as they arrive on the
 * blockchain. This is helpful for real-time applications such as notifications
 * or live-updating interfaces.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/real-time-transaction-processing}
 * @module transactionFeedsAPI
 */
const transactionFeedsAPI = (client) => {
  return {
    /**
     * Create a new transaction feed.
     *
     * @param {Object} params Parameters for creating Tansaction Feeds.
     * @param {String} params.alias A unique alias for the transaction feed.
     * @param {String} params.filter A valid filter string for the `/list-transactions`
     *                               endpoint. The transaction feed will be composed of future
     *                               transactions that match the filter.
     * @returns {TransactionFeed}
     */
    create: (params, cb) => {
      let body = Object.assign({ clientToken: uuid.v4() }, params)
      return shared.tryCallback(
        client.request('/create-transaction-feed', body).then(data => new TransactionFeed(data, client)),
        cb
      )
    },

    /**
     * Get single transaction feed given an id/alias.
     *
     * @param {Object} params Parameters to get single Tansaction Feed.
     * @param {String} params.id The unique ID of a transaction feed. Either `id` or
     *                           `alias` is required.
     * @param {String} params.alias The unique alias of a transaction feed. Either `id` or
     *                              `alias` is required.
     * @returns {TransactionFeed}
     */
    get: (params, cb) => shared.tryCallback(
      client.request('/get-transaction-feed', params).then(data => new TransactionFeed(data, client)),
      cb
    ),

    /**
     * Delete a transaction feed given an id/alias.
     *
     * @param {Object} params Parameters to delete single Tansaction Feed.
     * @param {String} params.id The unique ID of a transaction feed. Either `id` or
     *                           `alias` is required.
     * @param {String} params.alias The unique alias of a transaction feed. Either `id` or
     *                              `alias` is required.
     */
    delete: (params, cb) => shared.tryCallback(
      client.request('/delete-transaction-feed', params).then(data => data),
      cb
    ),


    /**
     * Returns a page of transaction feeds defined on the core.
     */
    query: (params, cb) => shared.query(client, this, '/list-transaction-feeds', params, {cb}),
  }
}

module.exports = transactionFeedsAPI
