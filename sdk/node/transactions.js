const errors = require('./errors')

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

class TransactionBuilder {
  constructor() {
    this.actions = []
  }

  issue(params) {
    this.actions.push(Object.assign({}, params, {type: 'issue'}))
  }

  controlWithAccount(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_account'}))
  }

  controlWithProgram(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_program'}))
  }

  spendFromAccount(params) {
    this.actions.push(Object.assign({}, params, {type: 'spend_account'}))
  }

  spendUnspentOutput(params) {
    this.actions.push(Object.assign({}, params, {type: 'spend_account_unspent_output'}))
  }

  retire(params) {
    this.actions.push(Object.assign({}, params, {type: 'control_program', control_program: '6a'}))
  }
}

module.exports = (client) => {
  return {
    build: (builderBlock) => {
      const builder = new TransactionBuilder()
      builderBlock(builder)

      return client.request('/build-transaction', [builder])
        .then(resp => checkForError(resp[0]))
    },
    submit: (signed) => {
      return client.request('/submit-transaction', {transactions: [signed]})
        .then(resp => checkForError(resp[0]))
    }
  }
}
