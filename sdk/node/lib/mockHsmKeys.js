const uuid = require('uuid')

/**
 * MockHsm
 * @class
 */
class MockHsmKeys {
  /**
   * constructor - return MockHsmKeys object configured for specified Chain Core
   *
   * @param  {Client} client Configured Chain client object
   */
  constructor(client) {

    /**
     * Create a new MockHsm key.
     *
     * @param {Object} [params={}] - Parameters for access token creation.
     * @param {string} params.alias - User specified, unique identifier.
     */
    this.create = (params = {}) => {
      let body = Object.assign({ client_token: uuid.v4() }, params)
      return client.request('/mockhsm/create-key', body)
        .then(data => data)
    }
  }
}

module.exports = MockHsmKeys
