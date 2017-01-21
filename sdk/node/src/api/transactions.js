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
   * @param {String} params.asset_id - Asset ID specifiying the asset to be issued.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.asset_alias - Asset alias specifying the asset to be issued.
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
   * @option params [String] :assetId Asset ID specifiying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.accountId - Account ID specifiying the account controlling the asset.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.accountAlias - Account alias specifying the account controlling the asset.
   *                                   You must specify either an ID or an alias.
   * @param {Number} params.amount - Amount of the asset to be controlled.
   */
  controlWithAccount(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_account'}))
  }

  /**
   * Add an action that controls assets with a control program.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.assetId - Asset ID specifiying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be controlled.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.controlProgram - The control program to be used.
   * @param {Number} params.amount - Amount of the asset to be controlled.
   */
  controlWithProgram(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_program'}))
  }

  /**
   * Add an action that spends assets from an account specified by identifier.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.assetId - Asset ID specifiying the asset to be spent.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be spent.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.accountId - Account ID specifiying the account spending the asset.
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
   * Add an action that spends an unspent output.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.transactionId - Transaction ID specifying the
   *                                        transaction to select an output from.
   * @param {Number} params.position - Position of the output within the
   *                                   transaction to be spent.
   */
  spendUnspentOutputDeprecated(params) {
    this.actions.push(Object.assign({}, params, {type: 'spend_account_unspent_output'}))
  }

  /**
   * Add an action that retires units of an asset.
   *
   * @param {Object} params - Action parameters.
   * @param {String} params.assetId - Asset ID specifiying the asset to be retired.
   *                                   You must specify either an ID or an alias.
   * @param {String} params.assetAlias - Asset alias specifying the asset to be retired.
   *                                   You must specify either an ID or an alias.
   * @param {Number} params.amount - Amount of the asset to be retired.
   */
  retire(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_program', controlProgram: '6a'}))
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
 * Processing callback for building a transaction. The instance of
 * {@link TransactionBuilder} modified in the function is used to build a transaction
 * in Chain Core.
 *
 * @callback Transactions~builderCallback
 * @param {TransactionBuilder} builder
 */

/**
 * A blockchain consists of an immutable set of cryptographically linked
 * transactions. Each transaction contains one or more actions.
 * <br/><br/>
 * More info: {@link https://chain.com/docs/core/build-applications/transaction-basics}
 * @module TransactionsApi
 */
const transactionsAPI = (client) => {
  return {
    /**
     * Get one page of transactions matching the specified query.
     *
     * @param {Query} params={} - Filter and pagination information.
     * @param {pageCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Page>} Requested page of results.
     */
    query: (params, cb) => shared.query(client, 'transactions', '/list-transactions', params, {cb}),

    /**
     * Request all transactions matching the specified query, calling the
     * supplied processor callback with each item individually.
     *
     * @param {Query} params - Filter and pagination information.
     * @param {QueryProcessor} processor - Processing callback.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise} A promise resolved upon processing of all items, or
     *                   rejected on error.
     */
    queryAll: (params, processor, cb) => shared.queryAll(client, 'transactions', params, processor, cb),

    /**
     * Build an unsigned transaction from a set of actions.
     *
     * @param {builderCallback} builderBlock - Function that adds desired actions
     *                                         to a given builder object.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<Object>} - Unsigned transaction template, or error.
     */
    build: (builderBlock, cb) => {
      const builder = new TransactionBuilder()
      builderBlock(builder)

      return shared.tryCallback(
        client.request('/build-transaction', [builder]).then(resp => checkForError(resp[0])),
        cb
      )
    },

    /**
     * Build multiple unsigned transactions from multiple sets of actions.
     *
     * @param {Array<builderCallback>} builderBlocks - Functions that add desired actions
     *                                                 to a given builder object, one
     *                                                 per transaction.
     * @param {objectCallback} [callback] - Optional callback. Use instead of Promise return value as desired.
     * @returns {Promise<BatchResponse>} - Batch of unsigned transaction templates, or errors.
     */
    buildBatch: (builderBlocks, cb) => {
      const builders = builderBlocks.map((builderBlock) => {
        const builder = new TransactionBuilder()
        builderBlock(builder)
        return builder
      })

      return shared.createBatch(client, '/build-transaction', builders, {cb})
    },

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
     * @returns {Promise<BatchResponse>} - Batch response of transaction IDs, or errors.
     */
    submitBatch: (signed, cb) => shared.tryCallback(
      client.request('/submit-transaction', {transactions: signed})
            .then(resp => new shared.BatchResponse(resp)),
      cb
    ),
  }
}

module.exports = transactionsAPI
