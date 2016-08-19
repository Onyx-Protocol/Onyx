import Client from './client'

class Context {
  constructor(config) {
    this.config = Object.assign({}, config)
    this.client = new Client(config.url)
  }
}

export default Context
