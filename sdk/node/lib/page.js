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
