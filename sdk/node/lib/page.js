class Page {
  constructor(data) {
    Object.assign(this, data)
  }

  nextPage() {
    // return this.itemClass.query(context, this.next)
  }

  [Symbol.iterator]() {
    const self = this

    return {
      index: 0,
      next: function() {
        if (this.index >= self.items.length) {
          return {done: true}
        } else {
          return { value: self.items[this.index++], done: false }
        }
      }
    }
  }
}

module.exports = Page
