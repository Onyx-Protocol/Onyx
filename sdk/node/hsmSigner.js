const Client = require('./client')

class HsmSigner {
  constructor() {
    this.signerConnections = {}
  }

  addKey(xpub, url, token = '') {
    const id = [url,token].join('-')
    let connection = this.signerConnections[id]
    if (!connection) {
      connection = this.signerConnections[id] = {
        connection: new Client(url, token),
        xpubs: []
      }
    }

    connection.xpubs.push(xpub)
  }

  sign(template) {
    let promise = Promise.resolve(template)

    if (Object.keys(this.signerConnections).length == 0) {
      return promise.then(() => template)
    }

    for (let signerId in this.signerConnections) {
      const signer = this.signerConnections[signerId]

      promise = promise.then(nextTemplate =>
        signer.connection.request('/sign-transaction', {
          transactions: [nextTemplate],
          xpubs: signer.xpubs
      })).then(resp => resp[0])
    }

    return promise
  }
}

module.exports = HsmSigner
