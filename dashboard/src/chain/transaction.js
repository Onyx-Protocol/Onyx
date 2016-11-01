import buildClass from './buildClass'
import errors from './errors'

class Transaction extends buildClass('transaction') {
  checkForError(data) {
    if ('code' in data) {
      throw errors.create(
        errors.types.BAD_REQUEST,
        errors.formatErrMsg(data, ''),
        data
      )
    }
    return data
  }

  build(context) {
    let body = [this]
    return context.client.request('/build-transaction', body)
      .then(data => this.checkForError(data[0]))
  }

  submit(context) {
    return this.constructor.submit([this], context)
      .then(data => this.checkForError(data[0]))
  }

  static submit(signedTransactions, context) {
    let body = {transactions: signedTransactions}
    return context.client.request('/submit-transaction', body)
      .then(data => data.map((item) => new Transaction(item)))
  }
}

delete Transaction.create
delete Transaction.prototype.create

export default Transaction
