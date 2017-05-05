const shared = require('../shared')

/**
 * API for interacting with Chain Ivy compiler
 *
 * @module IvyCompiler
 */
const ivyAPI = (client) => {
  return {
    /**
     * Compile Ivy source code.
     *
     * @param {Object} params - Options for source compilation.
     * @param {String} params.contract - The contract source.
     * @param {Array<Object>} param.args - Optional list of arguments used to instantiate the contract.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<CoreInfo>} Requested info of specified Chain Core.
     */
    compile: (params, cb) => shared.tryCallback(
      client.request('/compile', params),
      cb
    )
  }
}

module.exports = ivyAPI
