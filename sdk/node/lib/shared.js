const uuid = require('uuid')
const errors = require('./errors')
const Page = require('./page')
const Promise = require('./promiseWithCallback')

module.exports = {

  create: (client, path, params = {}, opts = {}) => {
    const object = Object.assign({ client_token: uuid.v4() }, params)
    let body = object
    if (!opts.skipArray) {
      body = [body]
    }

    return client.request(path, body).then(data => {
      if (errors.isBatchError(data[0])) {
        throw errors.newBatchError(data[0])
      }

      if (typeof data === 'Array') {
        return data[0]
      } else {
        return data
      }
    })
    .callback(opts.cb)
  },

  createBatch: (client, path, params = [], opts = {}) => {
    params = params.map((item) =>
      Object.assign({ client_token: uuid.v4() }, item))

    return client.request(path, params).then(resp => {
      return {
        successes: resp.map((item) => item.code ? null : item),
        errors: resp.map((item) => item.code ? item : null),
        response: resp,
      }
    })
  },

  query: (client, owner, path, params = {}, opts = {}) => {
    return client.request(path, params)
      .then(data => new Page(data, owner))
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

          if (!page.last_page) {
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
}
