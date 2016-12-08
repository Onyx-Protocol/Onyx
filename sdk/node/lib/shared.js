const uuid = require('uuid')
const errors = require('./errors')
const Page = require('./page')

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
  },

  createBatch : (client, path, params = []) => {
    params = params.map((item) =>
      Object.assign({ client_token: uuid.v4() }, item))

    return client.request(path, params).then(response => {
      return {
        successes: response.map((item) => item.code ? null : item),
        errors: response.map((item) => item.code ? item : null),
        response: response,
      }
    })
  },

  query: (client, path, params = {}) => {
    return client.request(path, params)
      .then(data => new Page(data))
  }
}
