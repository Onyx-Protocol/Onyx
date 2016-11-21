class Page {
  constructor(data, itemClass) {
    Object.assign(this, data)
    this.itemClass = itemClass
    this.items = (this.items || []).map((data) => {
      return new itemClass(data)
    })
  }

  nextPage(context) {
    return this.itemClass.query(context, this.next)
  }
}

export default Page
