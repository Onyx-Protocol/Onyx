const shared = require('../shared')

const uuid = require('uuid')

/**
 * Hardcoding value of (2 ** 63) - 1 since JavaScript rounds this value up,
 * which causes issues when attempting to query TransactionFeed.
 * @ignore
 */
const MAX_BLOCK_HEIGHT = '9223372036854775807'

/**
 * @class
 * A single transaction feed that can be consumed. See {@link TransactionFeeds}
 * for actions to create TransactionFeed objects.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/real-time-transaction-processing}
 *
 * @property {String} id
 * Unique transaction feed identifier.
 *
 * @property {String} alias
 * User specified, unique identifier.
 *
 * @property {String} filter
 * @property {String} after
 */
class TransactionFeed {
  /**
   * Called once for every item received via the transaction feed.
   *
   * @callback FeedProcessor
   * @param {Object} item - Item to process.
   * @param {function(Boolean)} next - Continue to the next item when it becomes
   *                                   available. Passing true to this callback
   *                                   will update the feed to acknowledge that
   *                                   the current item was consumed.
   * @param {function(Boolean)} done - Terminate the processing loop. Passing
   *                                   true to this callback will update the
   *                                   feed to acknowledge that the current item
   *                                   was consumed.
   * @param {function(Error)} fail - Terminate the processing loop due to an
   *                                 application-level error. This callback
   *                                 accepts an optional error argument. The
   *                                 feed will not be updated, and the current
   *                                 item will not be acknowledged.
   */

  /**
   * Create a new transaction feed consumer.
   *
   * @param {Object} feed - API response from {@link module:TransactionFeedsApi}
   *                        `create` or `get` call.
   * @param {Client} client - Configured Chain client object
   * @returns {TransactionFeed}
   */
  constructor(feed, client) {
    this.id = feed['id']
    this.alias = feed['alias']
    this.after = feed['after']
    this.filter = feed['filter']

    let nextAfter

    const ack = () => client.request('/update-transaction-feed', {
      id: this.id,
      after: nextAfter,
      previousAfter: this.after
    }).then(() => { this.after = nextAfter })

    const query = params => client.transactions.query(params)

    /**
     * Process items returned from a transaction feed in real-time.
     *
     * @param {FeedProcessor} consumer - Called once with each item to do any
     *                                   desired processing. The callback can
     *                                   optionally choose to terminate the loop.
     * @param {Number} [timeout=86400] - Number of seconds to wait before
     *                                   closing connection.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     */
    this.consume = (consumer, ...args) => {
      let timeout = 24*60*60
      let cb
      switch (args.length) {
        case 0:
          // promise with default timeout
          break
        case 1:
          if (args[0] instanceof Function) {
            cb = args[0]
          } else {
            timeout = args[0]
          }
          break
        case 2:
          timeout = args[0]
          cb = args[1]
          break
        default:
          throw new Error('Invalid arguments')
      }

      const promise = new Promise((resolve, reject) => {
        let queryArgs = {
          filter: this.filter,
          after: this.after,
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
                // - next(shouldAck): maybe ack, then continue/long-poll to next item.
                // - done(shouldAck): maybe ack, then terminate the loop by fulfilling the outer promise.
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

      return shared.tryCallback(promise, cb)
    }
  }
}

/**
 * You can use transaction feeds to process transactions as they arrive on the
 * blockchain. This is helpful for real-time applications such as notifications
 * or live-updating interfaces.
 * 
 * More info: {@link https://chain.com/docs/core/build-applications/real-time-transaction-processing}
 * @module TransactionFeedsApi
 */
const transactionFeedsAPI = (client) => {
  return {
    /**
     * Create a new transaction feed.
     *
     * @param {Object} params - Parameters for creating Transaction Feeds.
     * @param {String} params.alias - A unique alias for the transaction feed.
     * @param {String} params.filter - A valid filter string for the `/list-transactions`
     *                               endpoint. The transaction feed will be composed of future
     *                               transactions that match the filter.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<TransactionFeed>} Newly created transaction feed
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
     * @param {Object} params - Parameters to get single Transaction Feed.
     * @param {String} params.id - The unique ID of a transaction feed. Either `id` or
     *                           `alias` is required.
     * @param {String} params.alias - The unique alias of a transaction feed. Either `id` or
     *                              `alias` is required.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<TransactionFeed>} Requested transaction feed object
     */
    get: (params, cb) => shared.tryCallback(
      client.request('/get-transaction-feed', params).then(data => new TransactionFeed(data, client)),
      cb
    ),

    /**
     * Delete a transaction feed given an id/alias.
     *
     * @param {Object} params - Parameters to delete single Transaction Feed.
     * @param {String} params.id - The unique ID of a transaction feed. Either `id` or
     *                           `alias` is required.
     * @param {String} params.alias - The unique alias of a transaction feed. Either `id` or
     *                              `alias` is required.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @return {Promise} Promise resolved on success
     */
    delete: (params, cb) => shared.tryCallback(
      client.request('/delete-transaction-feed', params).then(data => data),
      cb
    ),


    /**
     * Get one page of transaction feeds.
     *
     * @param {Object} params={} - Pagination information.
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<TransactionFeed>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'transactionFeeds', '/list-transaction-feeds', params, {cb}),

    /**
     * Request all transaction feeds matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Object} params={} - Pagination information.
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {QueryProcessor<TransactionFeed>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'transactionFeeds', params, processor, cb),
  }
}

module.exports = transactionFeedsAPI
