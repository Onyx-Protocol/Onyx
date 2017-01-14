const uuid = require('uuid')
const errors = require('./errors')
const Page = require('./page')

/**
 * @callback objectCallback
 * @param {error} error
 * @param {Object} object - Object response from API.
 */

/**
 * @callback batchCallback
 * @param {error} error
 * @param {BatchResponse} batchResponse - Newly created objects (and errors).
 */

/**
 * Object specifying how to request records from a given endpoint. Queries can
 * be optionally extended with additional fields to provide extra options for
 * filtering.
 *
 * @typedef {Object} Query
 * @property {String} [filter] - String used to filter results. See the
 *                            {@link https://chain.com/docs/core/build-applications/queries#filters|documentation on filter strings}
 *                            for more details
 * @property {String} [after] - Cursor pointing to the start of the result set
 * @property {Number} [pageSize] - Number of items to return in result set
 */

/**
 * Called once for each item in the result set.
 *
 * @callback QueryProcessor
 * @param {Object} item - Item to process
 */

/**
 * @class
 */
class BatchResponse {
  /**
   * constructor
   *
   * @param  {Array<Object>} resp - List of items which are objects or errors
   */
  constructor(resp) {
    /**
     * Items from the input array which were successfully processed
     * @type {Array<Object>}
     */
    this.successes = resp.map((item) => item.code ? null : item)

    /**
     * Items from the input array which reuslted in an error
     * @type {Array<Object>}
     */
    this.errors = resp.map((item) => item.code ? item : null)

    /**
     * Original input array
     * @type {Array<Object>}
     */
    this.response = resp
  }
}

const tryCallback = (promise, cb) => {
  if (typeof cb !== 'function') return promise

  return promise.then(value => {
    setTimeout(() => cb(null, value), 0)
  }, error => {
    setTimeout(() => cb(error, null), 0)
  })
}

module.exports = {
  create: (client, path, params = {}, opts = {}) => {
    const object = Object.assign({ clientToken: uuid.v4() }, params)
    let body = object
    if (!opts.skipArray) {
      body = [body]
    }

    return tryCallback(
      client.request(path, body).then(data => {
        if (errors.isBatchError(data[0])) throw errors.newBatchError(data[0])

        if (Array.isArray(data)) return data[0]
        return data
      }),
      opts.cb
    )
  },

  createBatch: (client, path, params = [], opts = {}) => {
    params = params.map((item) =>
      Object.assign({ clientToken: uuid.v4() }, item))

    return tryCallback(
      client.request(path, params).then(resp => new BatchResponse(resp)),
      opts.cb
    )
  },

  query: (client, owner, path, params = {}, opts = {}) => {
    return tryCallback(
      client.request(path, params).then(data => new Page(data, owner)),
      opts.cb
    )
  },

  /*
   * NOTE: Requires query to be implemented on `owner` object
   */
  queryAll: (owner, params, processor = () => {}) => {
    let nextParams = params

    return new Promise((resolve, reject) => {
      const nextPage = () => {
        owner.query(nextParams).then(page => {
          for (let item in page.items) {
            processor(page.items[item])
          }

          if (!page.lastPage) {
            nextParams = page.next
            nextPage()
            return
          } else {
            resolve()
          }
        }).catch(reject)
      }

      nextPage()
    })
  },

  tryCallback,
  BatchResponse,
}
