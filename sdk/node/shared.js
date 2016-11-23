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
  query: (client, path, params = {}) => {
    // console.log(this)
    // console.log(this.constructor)
    //
    // console.log(path)
    // console.log(params)
    return client.request(path, params)
      .then(data => new Page(data))
  }
}
