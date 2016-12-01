const uuid = require('uuid')
const errors = require('./errors')
const Page = require('./page')

module.exports = {
  create: (client, path, params = {}) => {
    let object = Object.assign({ client_token: uuid.v4() }, params)

    return client.request(path, [object]).then(data => {
      if (errors.isBatchError(data[0])) {
        throw errors.newBatchError(data[0])
      }

      return data[0]
    })
  },

  createBatch : (client, path, params = []) => {
    params = params.map((item) =>
      Object.assign({ client_token: uuid.v4() }, item))

    return client.request(path, params).then(response => {
      return {
        successes: response.filter((item) => !item.code),
        errors: response.filter((item) => item.code),
        response: response,
      }
    })
  },

  query: (client, path, params = {}) => {
    return client.request(path, params)
      .then(data => new Page(data))
  }
}
