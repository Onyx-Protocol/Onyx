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
 * Called once for each item in the result set.
 *
 * @callback QueryProcessor
 * @param {Object} item - Item to process.
 * @param {function} next - Call to proceed to the next item for processing.
 * @param {function(err)} done - Call to terminate iteration through the result
 *                               set. Accepts an optional error argument which
 *                               will be passed to the promise rejection or
 *                               callback depending on async calling style.
 */

 /**
  * @typedef {Object} Key
  * @global
  *
  * @property {String} rootXpub
  * @property {String} accountXpub
  * @property {String[]} accountDerivationPath
  */

/**
 * @class
 */
class BatchResponse {
  /**
   * constructor
   *
   * @param  {Array<Object>} resp - List of items which are objects or errors.
   */
  constructor(resp) {
    /**
     * Items from the input array which were successfully processed. This value
     * is a sparsely populated array, maintaining the indexes of the items as
     * they were originally submitted.
     * @type {Array<Object>}
     */
    this.successes = []

    /**
     * Items from the input array which resulted in an error. This value
     * is a sparsely populated array, maintaining the indexes of the items as
     * they were originally submitted.
     * @type {Array<Object>}
     */
    this.errors = []

    resp.forEach((item, index) => {
      if (item.code) {
        this.errors[index] = item
      } else {
        this.successes[index] = item
      }
    })

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

const batchRequest = (client, path, params, cb) => {
  return tryCallback(
    client.request(path, params).then(resp => new BatchResponse(resp)),
    cb
  )
}

module.exports = {
  batchRequest,

  singletonBatchRequest: (client, path, params = {}, cb) => {
    return tryCallback(
      batchRequest(client, path, [params]).then(batch => {
        if (batch.errors[0]) {
          throw errors.newBatchError(batch.errors[0])
        }
        return batch.successes[0]
      }),
      cb
    )
  },

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

  query: (client, memberPath, path, params = {}, opts = {}) => {
    return tryCallback(
      client.request(path, params).then(data => new Page(data, client, memberPath)),
      opts.cb
    )
  },

  /*
   * NOTE: Requires query to be implemented on client for the specified member.
   */
  queryAll: (client, memberPath, params, processor = () => {}, cb) => {
    let nextParams = params

    let queryOwner = client
    memberPath.split('.').forEach((member) => {
      queryOwner = queryOwner[member]
    })

    const promise = new Promise((resolve, reject) => {
      const done = (err) => {
        if (cb) {
          cb(err)
          return
        } else if (err) {
          reject(err)
        }

        resolve()
      }

      const nextPage = () => {
        queryOwner.query(nextParams).then(page => {
          let index = 0
          let item

          const next = () => {
            if (index >= page.items.length) {
              if (page.lastPage) {
                done()
              } else {
                nextParams = page.next
                nextPage()
              }
              return
            }

            item = page.items[index]
            index++

            // Pass the next item to the processor, as well as two loop
            // operations:
            //
            // - next(): Continue to next item
            // - done(err): Then terminate the loop by fulfilling the outer promise
            //
            // The process can also terminate the loop by returning a promise
            // that will reject.

            let res = processor(item, next, done)
            if (res && typeof res.catch === 'function') {
              res.catch(reject)
            }
          }

          next()
        }).catch(reject) // fail processor loop on query failure
      }

      nextPage()
    })

    return tryCallback(promise, cb)
  },

  tryCallback,
  BatchResponse,
}
