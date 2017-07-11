const shared = require('../shared')
const errors = require('../errors')

// TODO: replace with default handler in requestSingle/requestBatch variants
function checkForError(resp) {
  if ('code' in resp) {
    throw errors.create(
      errors.types.BAD_REQUEST,
      errors.formatErrMsg(resp, ''),
      {body: resp}
    )
  }
  return resp
}

/**
 * A blockchain consists of an immutable set of cryptographically linked
 * transactions. Each transaction contains one or more actions.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/transaction-basics}
 * @typedef {Object} Transaction
 * @global
 *
 * @property {String} id
 * Unique transaction identifier.
 *
 * @property {String} timestamp
 * Time of transaction, RFC3339 formatted.
 *
 * @property {String} blockId
 * Unique identifier, or block hash, of the block containing a transaction.
 *
 * @property {Number} blockHeight
 * Height of the block containing a transaction.
 *
 * @property {Number} position
 * Position of a transaction within the block.
 *
 * @property {Object} referenceData
 * User specified, unstructured data embedded within a transaction.
 *
 * @property {Boolean} isLocal
 * A flag indicating one or more inputs or outputs are local.
 *
 * @property {TransactionInput[]} inputs
 * List of specified inputs for a transaction.
 *
 * @property {TransactionOutput[]} outputs
 * List of specified outputs for a transaction.
 */

/**
 * @typedef {Object} TransactionInput
 * @global
 *
 * @property {String} type
 * The type of the input. Possible values are "issue", "spend".
 *
 * @property {String} assetId
 * The id of the asset being issued or spent.
 *
 * @property {String} assetAlias
 * The alias of the asset being issued or spent (possibly null).
 *
 * @property {Hash} assetDefinition
 * The definition of the asset being issued or spent (possibly null).
 *
 * @property {Hash} assetTags
 * The tags of the asset being issued or spent (possibly null).
 *
 * @property {Boolean} assetIsLocal
 * A flag indicating whether the asset being issued or spent is local.
 *
 * @property {Integer} amount
 * The number of units of the asset being issued or spent.
 *
 * @property {String} spentOutputId
 * The id of the output consumed by this input. ID is nil if this is an issuance input.
 *
 * @property {String} accountId
 * The id of the account transferring the asset (possibly null if the
 * input is an issuance or an unspent output is specified).
 *
 * @property {String} accountAlias
 * The alias of the account transferring the asset (possibly null if the
 * input is an issuance or an unspent output is specified).
 *
 * @property {String} accountTags
 * The tags associated with the account (possibly null).
 *
 * @property {String} issuanceProgram
 * A program specifying a predicate for issuing an asset (possibly null
 * if input is not an issuance).
 *
 * @property {Object} referenceData
 * User specified, unstructured data embedded within an input (possibly null).
 *
 * @property {Boolean} isLocal
 * A flag indicating if the input is local.
 */

/**
 * Each new transaction in the blockchain consumes some unspent outputs and
 * creates others. An output is considered unspent when it has not yet been used
 * as an input to a new transaction. All asset units on a blockchain exist in
 * the unspent output set.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/unspent-outputs}
 * @typedef {Object} TransactionOutput
 * @global
 *
 * @property {String} id
 * The id of the output.
 *
 * @property {String} type
 * The type of the output. Possible values are "control" and "retire".
 *
 * @property {String} purpose
 * The purpose of the output. Possible values are "receive" and "change".
 *
 * @property {Number} position
 * The output's position in a transaction's list of outputs.
 *
 * @property {String} assetId
 * The id of the asset being issued or spent.
 *
 * @property {String} assetAlias
 * The alias of the asset being issued or spent (possibly null).
 *
 * @property {Hash} assetDefinition
 * The definition of the asset being issued or spent (possibly null).
 *
 * @property {Hash} assetTags
 * The tags of the asset being issued or spent (possibly null).
 *
 * @property {Boolean} assetIsLocal
 * A flag indicating whether the asset being issued or spent is local.
 *
 * @property {Integer} amount
 * The number of units of the asset being issued or spent.
 *
 * @property {String} accountId
 * The id of the account transferring the asset (possibly null).
 *
 * @property {String} accountAlias
 * The alias of the account transferring the asset (possibly null).
 *
 * @property {String} accountTags
 * The tags associated with the account (possibly null).
 *
 * @property {String} controlProgram
 * The control program which must be satisfied to transfer this output.
 *
 * @property {Object} referenceData
 * User specified, unstructured data embedded within an input (possibly null).
 *
 * @property {Boolean} isLocal
 * A flag indicating if the input is local.
 */

/**
 * @class
 * A convenience class for building transaction template objects.
 */
class TransactionBuilder {
  /**
   * constructor - return a new object used for constructing a transaction.
   */
  constructor() {
    this.actions = []


    /**
     * If true, build the transaction as a partial transaction.
     * @type {Boolean}
     */
    this.allowAdditionalActions = false

    /**
     * Base transaction provided by a third party.
     * @type {Object}
     */
    this.baseTransaction = null
  }

  /**
   * Add an action that issues assets.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.assetId - Asset ID specifying the asset to be issued.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be issued.
   *                                      You must specify either an ID or an alias.
   * @param {String} params.amount - Amount of the asset to be issued.
   */
  issue(params) {
    this.actions.push(Object.assign({}, params, {type: 'issue'}))
  }

  /**
   * Add an action that controls assets with an account specified by identifier.
   *
   * @param {Object} params - Action parameters.
   * @option params [String] :assetId Asset ID specifying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.accountId - Account ID specifying the account controlling the asset.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.accountAlias - Account alias specifying the account controlling the asset.
   *                                   You must specify either an ID or an alias.
   * @param {Number} params.amount - Amount of the asset to be controlled.
   */
  controlWithAccount(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_account'}))
  }

  /**
   * Add an action that controls assets with a receiver.
   *
   * @param {Object} params - Action parameters.
   * @param {Object} params.receiver - The receiver object in which assets will be controlled.
   * @param {String} params.assetId - Asset ID specifying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {Number} params.amount - Amount of the asset to be controlled.
   */
  controlWithReceiver(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_receiver'}))
  }

  /**
   * Add an action that spends assets from an account specified by identifier.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.assetId - Asset ID specifying the asset to be spent.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be spent.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.accountId - Account ID specifying the account spending the asset.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.accountAlias - Account alias specifying the account spending the asset.
   *                                   You must specify either an ID or an alias.
   * @param {Number} params.amount - Amount of the asset to be spent.
   */
  spendFromAccount(params) {
    this.actions.push(Object.assign({}, params, {type: 'spend_account'}))
  }

  /**
   * Add an action that spends an unspent output.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.outputId - ID of the transaction output to be spent.
   */
  spendUnspentOutput(params) {
    this.actions.push(Object.assign({}, params, {type: 'spend_account_unspent_output'}))
  }

  /**
   * Add an action that retires units of an asset.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.assetId - Asset ID specifying the asset to be retired.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be retired.
   *                                   You must specify either an ID or an alias.
   * @param {Number} params.amount - Amount of the asset to be retired.
   */
  retire(params) {
    this.actions.push(Object.assign({}, params, {type: 'retire'}))
  }

  /**
   * transactionReferenceData - Sets the transaction-level reference data. May
   *                            only be used once per transaction.
   *
   * @param {Object} referenceData - User specified, unstructured data to
   *                                  be embedded in a transaction.
   */
  transactionReferenceData(referenceData) {
    this.actions.push({
      type: 'set_transaction_reference_data',
      referenceData
    })
  }
}

/**
 * API for interacting with {@link Transaction transactions}.
 *
 * More info: {@link https://chain.com/docs/core/build-applications/transaction-basics}
 * @module TransactionsApi
 */
const transactionsAPI = (client) => {
  /**
   * Processing callback for building a transaction. The instance of
   * {@link TransactionBuilder} modified in the function is used to build a transaction
   * in Chain Core.
   *
   * @callback builderCallback
   * @param {TransactionBuilder} builder
   */

  // TODO: implement finalize
  const finalize = (template, cb) => shared.tryCallback(
    Promise.resolve(template),
    cb
  )

  // TODO: implement finalizeBatch
  const finalizeBatch = (templates, cb) => shared.tryCallback(
    Promise.resolve(new shared.BatchResponse(templates)),
    cb
  )

  return {
    /**
     * Get one page of transactions matching the specified query.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Number} params.startTime -  A Unix timestamp in milliseconds. When specified, only transactions with a block time greater than the start time will be returned.
     * @param {Number} params.endTime - A Unix timestamp in milliseconds. When specified, only transactions with a block time less than the start time will be returned.
     * @param {Number} params.timeout - A time in milliseconds after which a server timeout should occur. Defaults to 1000 (1 second).
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page<Transaction>>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'transactions', '/list-transactions', params, {cb}),

    /**
     * Request all transactions matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Object} params={} - Filter and pagination information.
     * @param {String} params.filter - Filter string, see {@link https://chain.com/docs/core/build-applications/queries}.
     * @param {Array<String|Number>} params.filterParams - Parameter values for filter string (if needed).
     * @param {Number} params.startTime -  A Unix timestamp in milliseconds. When specified, only transactions with a block time greater than the start time will be returned.
     * @param {Number} params.endTime - A Unix timestamp in milliseconds. When specified, only transactions with a block time less than the start time will be returned.
     * @param {Number} params.timeout - A time in milliseconds after which a server timeout should occur. Defaults to 1000 (1 second).
     * @param {Number} params.pageSize - Number of items to return in result set.
     * @param {QueryProcessor<Transaction>} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'transactions', params, processor, cb),

    /**
     * Build an unsigned transaction from a set of actions.
     *
     * @param {module:TransactionsApi~builderCallback} builderBlock - Function that adds desired actions
     *                                         to a given builder object.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} Unsigned transaction template, or error.
     */
    build: (builderBlock, cb) => {
      const builder = new TransactionBuilder()

      try {
        builderBlock(builder)
      } catch (err) {
        return Promise.reject(err)
      }

      return shared.tryCallback(
        client.request('/build-transaction', [builder]).then(resp => checkForError(resp[0])),
        cb
      )
    },

    /**
     * Build multiple unsigned transactions from multiple sets of actions.
     *
     * @param {Array<module:TransactionsApi~builderCallback>} builderBlocks - Functions that add desired actions
     *                                                 to a given builder object, one
     *                                                 per transaction.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<BatchResponse>} Batch of unsigned transaction templates, or errors.
     */
    buildBatch: (builderBlocks, cb) => {
      const builders = []
      for (let i in builderBlocks) {
        const b = new TransactionBuilder()
        try {
          builderBlocks[i](b)
        } catch (err) {
          return Promise.reject(err)
        }
        builders.push(b)
      }

      return shared.createBatch(client, '/build-transaction', builders, {cb})
    },

    /**
     * sign - Sign a single transaction.
     *
     * @param {Object} template - A single transaction template.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Object} Transaction template with all possible signatures added.
     */
    sign: (template, cb) => finalize(template)
      .then(finalized => client.signer.sign(finalized, cb)),

    /**
     * signBatch - Sign a batch of transactions.
     *
     * @param {Array<Object>} templates Array of transaction templates.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {BatchResponse} Tranasaction templates with all possible signatures
     *                         added, as well as errors.
     */
    signBatch: (templates, cb) => finalizeBatch(templates)
      // TODO: merge batch errors from finalizeBatch
      .then(finalized => client.signer.signBatch(finalized.successes, cb)),

    /**
     * Submit a signed transaction to the blockchain.
     *
     * @param {Object} signed - A fully signed transaction template.
     * @returns {Promise<Object>} Transaction ID of the successful transaction, or error.
     */
    submit: (signed, cb) => shared.tryCallback(
      client.request('/submit-transaction', {transactions: [signed]}).then(resp => checkForError(resp[0])),
      cb
    ),

    /**
     * Submit multiple signed transactions to the blockchain.
     *
     * @param {Array<Object>} signed - An array of fully signed transaction templates.
     * @returns {Promise<BatchResponse>} Batch response of transaction IDs, or errors.
     */
    submitBatch: (signed, cb) => shared.tryCallback(
      client.request('/submit-transaction', {transactions: signed})
            .then(resp => new shared.BatchResponse(resp)),
      cb
    ),
  }
}

module.exports = transactionsAPI
