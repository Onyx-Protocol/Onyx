const shared = require('../shared')

/**
 * @class
 * In order to issue or transfer asset units on a blockchain, a transaction is
 * created in Chain Core and sent to the HSM for signing. The HSM signs the
 * transaction without ever revealing the private key. Once signed, the
 * transaction can be submitted to the blockchain successfully.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/keys}
 */
class HsmSigner {

  /**
   * Create a new HSM signer object.
   *
   * @returns {HsmSigner}
   */
  constructor() {
    this.signers = {}
  }

  /**
   * addKey - Add a new key/signer pair to the HSM signer.
   *
   * @param {Object|String} key - An object with an xpub key, or an xpub as a string.
   * @param {Connection} connection - Authenticated connection to a specific HSM instance.
   * @returns {void}
   */
  addKey(key, connection) {
    const id = `${connection.baseUrl}-${connection.token || 'noauth'}`
    let signer = this.signers[id]
    if (!signer) {
      signer = this.signers[id] = {
        connection: connection,
        xpubs: []
      }
    }

    signer.xpubs.push(typeof key == 'string' ? key : key.xpub)
  }

  /**
   * sign - Sign a single transaction.
   *
   * @param {Object} template - A single transaction template.
   * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @returns {Object} Transaction template with all possible signatures added.
   */
  sign(template, cb) {
    let promise = Promise.resolve(template)

    // Return early if no signers
    if (Object.keys(this.signers).length == 0) {
      return shared.tryCallback(promise, cb)
    }

    for (let signerId in this.signers) {
      const signer = this.signers[signerId]

      promise = promise.then(nextTemplate =>
        signer.connection.request('/sign-transaction', {
          transactions: [nextTemplate],
          xpubs: signer.xpubs
        })
      ).then(resp => resp[0])
    }

    return shared.tryCallback(promise, cb)
  }

  /**
   * signBatch - Sign a batch of transactions.
   *
   * @param {Array<Object>} templates Array of transaction templates.
   * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
   * @returns {BatchResponse} Tranasaction templates with all possible signatures
   *                         added, as well as errors.
   */
  signBatch(templates, cb) {
    templates = templates.filter((template) => template != null)
    let promise = Promise.resolve(templates)

    // Return early if no signers
    if (Object.keys(this.signers).length == 0) {
      return shared.tryCallback(promise.then(() => new shared.BatchResponse(templates)), cb)
    }

    let originalIndex = [...Array(templates.length).keys()]
    const errors = []

    for (let signerId in this.signers) {
      const nextTemplates = []
      const nextOriginalIndex = []
      const signer = this.signers[signerId]

      promise = promise.then(txTemplates =>
        signer.connection.request('/sign-transaction', {
          transactions: txTemplates,
          xpubs: signer.xpubs
        }).then(resp => {
          const batchResponse = new shared.BatchResponse(resp)

          batchResponse.successes.forEach((template, index) => {
            nextTemplates.push(template)
            nextOriginalIndex.push(originalIndex[index])
          })

          batchResponse.errors.forEach((error, index) => {
            errors[originalIndex[index]] = error
          })

          originalIndex = nextOriginalIndex
          return nextTemplates
        })
      )
    }

    return shared.tryCallback(promise.then(txTemplates => {
      const resp = []
      txTemplates.forEach((item, index) => {
        resp[originalIndex[index]] = item
      })

      errors.forEach((error, index) => {
        if (error != null) {
          resp[index] = error
        }
      })

      return new shared.BatchResponse(resp)
    }), cb)
  }
}

module.exports = HsmSigner
