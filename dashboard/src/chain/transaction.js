import buildClass from './buildClass'

class Transaction extends buildClass('transaction') {
  build(context) {
    let body = [this]
    return context.client.request('/build-transaction-template', body)
      .then(data =>  data[0])
  }

  static submit(signedTransactions, context) {
    let body = {transactions: signedTransactions}
    return context.client.request('/submit-transaction-template', body)
      .then(data => data.map((item) => {
        return new Transaction(item)
      }))
  }
}

delete Transaction.create
delete Transaction.prototype.create

export default Transaction
