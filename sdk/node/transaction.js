// TODO: replace with default handler in requestSingle/requestBatch variants
cosnt checkForError = (resp) => {
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
    this.actions = Object.assign({}, params, {type: 'issue'})
  }

  controlWithAccount(params) {
    this.actions = Object.assign({}, params, {type: 'control_with_account'})
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
