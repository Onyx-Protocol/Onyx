/**
 * Config
 * @class
 */
class Config {
  /**
   * @param  {Client} client Configured Chain client object.
   */
  constructor(client) {
    this.reset = (everything = false) => client.request('/reset', {everything: everything})

    this.configure = (opts = {}) => client.request('/configure', opts)

    this.info = () => client.request('/info')
  }
}

module.exports = Config
