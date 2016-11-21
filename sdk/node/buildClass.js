const uuid = require('uuid')

const Page = require('./page')
const errors = require('./errors')

function buildClass(type, options = {}) {
  const createPath = options.createPath || `/create-${type}`
  const listPath   = options.listPath || `/list-${type}s`

  return class {
    constructor(data) {
      Object.assign(this, data)
    }

    create(context) {
      let body = Object.assign({ client_token: uuid.v4() }, this)
      return this.constructor.create(context, [body]).then(data => {
        if (errors.isBatchError(data[0])) {
          throw errors.newBatchError(data[0])
        }
        return data[0]
      })
    }

    // NOTE: static create requires client_token to be set
    // by another method
    static create(context, opts) {
      return context.client.request(createPath, opts)
        .then(data => data.map((item) => new this(item)))
    }

    static query(context, opts) {
      return context.client.request(listPath, opts)
        .then(data => new Page(data, this))
    }

  }
}

module.exports = buildClass
