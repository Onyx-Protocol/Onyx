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
   * @param  {Object} data  API response for a single page of data.
   * @param  {Object} owner Chain API object implementing the `query` method.
   */
  constructor(data, owner) {
    /**
     * Array of Chain Core objects
     * @type {Array}
     */
    this.items = []

    /**
     * Object representing the query for the immediate next page of results.
     * @type {Query}
     */
    this.next = {}


    /**
     * Indicator that there are more results to load if true.
     * @type {Boolean}
     */
    this.lastPage = false

    Object.assign(this, data)

    this.owner = owner
  }

  /**
   * Fetch the next page of data for the query specified in this object.
   *
   * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @return {Promise<Page>} A promise resolving to a Page object containing
   *                         the requested results.
   */
  nextPage(cb) {
    return this.owner.query(this.next, cb)
  }
}

module.exports = Page
