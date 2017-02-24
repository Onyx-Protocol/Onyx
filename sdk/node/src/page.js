/**
 * @callback pageCallback
 * @param {error} error
 * @param {Page} page - Requested page of results.
 */

/**
 * @class
 * One page of results returned from an API request. With any given page object,
 * the next page of results in the query set can be requested.
 */
class Page {

  /**
   * Create a page object
   *
   * @param  {Object} data API response for a single page of data.
   * @param  {Client} client Chain Client.
   * @param  {String} memberPath key-path pointing to module implementing the
   *                  desired `query` method.
   */
  constructor(data, client, memberPath) {
    /**
     * Array of Chain Core objects. Available types are documented in the
     * {@link global global namespace}.
     *
     * @type {Array}
     */
    this.items = []

    /**
     * Object representing the query for the immediate next page of results. Can
     * be passed without modification to the `query` method that generated the
     * Page object containing it. 
     * @type {Object}
     */
    this.next = {}


    /**
     * Indicator that there are more results to load if true.
     * @type {Boolean}
     */
    this.lastPage = false

    Object.assign(this, data)

    this.client = client
    this.memberPath = memberPath
  }

  /**
   * Fetch the next page of data for the query specified in this object.
   *
   * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @returns {Promise<Page>} A promise resolving to a Page object containing
   *                         the requested results.
   */
  nextPage(cb) {
    let queryOwner = this.client
    this.memberPath.split('.').forEach((member) => {
      queryOwner = queryOwner[member]
    })

    return queryOwner.query(this.next, cb)
  }
}

module.exports = Page
