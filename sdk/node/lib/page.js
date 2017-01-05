/**
 * @callback pageCallback
 * @param {error} error
 * @param {Page} page - Requested page of results
 */

/**
 * @class
 */
class Page {
  constructor(data, owner) {
    Object.assign(this, data)

    this.owner = owner
  }

  nextPage() {
    return this.owner.query(this.next)
  }
}

module.exports = Page
