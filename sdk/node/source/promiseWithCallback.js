class PromiseWithCallback extends Promise {
  callback(cb) {
    if (typeof cb !== 'function') return this

    return this.then(value => {
      setTimeout(() => cb(null, value), 0)
    }, error => {
      setTimeout(() => cb(error, null), 0)
    })
  }
}

module.exports = PromiseWithCallback
