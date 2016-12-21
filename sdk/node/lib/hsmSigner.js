const Client = require('./client')

class HsmSigner {
  constructor() {
    this.signerConnections = {}
  }

  addKey(key, client) {
    const id = `${client.baseUrl}-${client.token || 'noauth'}`
    let connection = this.signerConnections[id]
    if (!connection) {
      connection = this.signerConnections[id] = {
        connection: client,
        xpubs: []
      }
    }

    console.log(typeof key);

    // connection.xpubs.push(xpub)
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

  signBatch(templates) {
    let promise = Promise.resolve(templates)

    if (Object.keys(this.signerConnections).length == 0) {
      return promise.then(() => templates)
    }

    for (let signerId in this.signerConnections) {
      const signer = this.signerConnections[signerId]

      promise = promise.then(nextTemplates =>
        signer.connection.request('/sign-transaction', {
          transactions: nextTemplates,
          xpubs: signer.xpubs
      })).then(resp => {
        return {
          successes: resp.filter((item) => !item.code),
          errors: resp.filter((item) => item.code),
          response: resp,
        }
      })
    }

    return promise
  }
}

module.exports = HsmSigner
