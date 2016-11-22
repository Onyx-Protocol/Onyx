const uuid = require('uuid')
const errors = require('./errors')

module.exports = {
  create: (client, path, params = {}) => {
    let object = Object.assign({ client_token: uuid.v4() }, params)

    return client.request(path, [object]).then(data => {
      if (errors.isBatchError(data[0])) {
        throw errors.newBatchError(data[0])
      }

      return data[0]
    })
  }
}
