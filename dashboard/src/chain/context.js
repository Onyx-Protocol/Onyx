import Client from './client'

class Context {
  constructor(config) {
    this.config = {...config}
    this.client = new Client(config.url, config.clientToken)
  }
}

export default Context
