import buildClass from './buildClass'
import errors from './errors'

class Transaction extends buildClass('transaction') {
  checkForError(resp) {
    if ('code' in resp) {
      throw errors.create(
        errors.types.BAD_REQUEST,
        errors.formatErrMsg(resp, ''),
        {body: resp}
      )
    }
    return resp
  }

  build(context) {
    let body = [this]
    return context.client.request('/build-transaction', body)
      .then(resp => this.checkForError(resp[0]))
  }

  submit(context) {
    return this.constructor.submit([this], context)
      .then(resp => this.checkForError(resp[0]))
  }

  static submit(signedTransactions, context) {
    let body = {transactions: signedTransactions}
    return context.client.request('/submit-transaction', body)
      .then(resp => resp.map((item) => new Transaction(item)))
  }
}

delete Transaction.create
delete Transaction.prototype.create

export default Transaction
