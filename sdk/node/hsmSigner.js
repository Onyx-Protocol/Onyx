const Client = require('./client')

class HsmSigner {
  constructor() {
    signerConnections = {}
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
    let promise = Promise.resolve()

    if (Object.keys(this.signerConnections).length == 0) {
      return promise.then(() => template)
    }

    for (var signer in this.signerConnections) {
      if (object.hasOwnProperty(signer)) {

      }
    }
  }
}

module.exports = HsmSigner
