const Client = require('./client')

class Context {
  constructor(config) {
    this.config = Object.assign({}, config)
    this.client = new Client(config.url, config.clientToken)
  }
}

module.exports = Context
