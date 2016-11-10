import buildClass from './buildClass'
import uuid from 'uuid'
import Transaction from './transaction'

export default class MockHsm extends buildClass('mockhsm', {
  listPath: '/mockhsm/list-keys'
}) {
  create(context) {
    let body = Object.assign({ client_token: uuid.v4() }, this)
    return context.client.request('/mockhsm/create-key', body)
      .then(data => new MockHsm(data))
  }

  // TODO: handle batch errors
  static sign(transactions, xpubs, context) {
    return context.client.request('/mockhsm/sign-transaction', {
      transactions: transactions,
      xpubs: xpubs
    }).then(data => data.map((item) => new Transaction(item)))
  }
}
