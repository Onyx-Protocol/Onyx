const uuid = require('uuid')
const shared = require('./shared')

/**
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

    /**
     * Get one page of MockHsm keys, optionally filtered to specified aliases
     *
     * @param {Array.<string>} [aliases] List of requested aliases, max 200
     * @returns {Promise<Page>} Requested page of results
     */
    this.query = (aliases = []) => {
      let params = {aliases}
      if (aliases.length > 0) {
        params.page_size = aliases.length
      }

      return shared.query(client, this, '/mockhsm/list-keys', params)
    }

    /**
     * Request all MockHsm keys matching the specified filter, calling the
     * supplied processor callback with each item individually.
     *
     * @param {QueryProcessor} processor Processing callback.
     * @return {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error
     */
    this.queryAll = (processor) => shared.queryAll(this, {}, processor)
  }
}

module.exports = MockHsmKeys
