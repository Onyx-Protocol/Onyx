const uuid = require('uuid')

// const Transaction = require('./transaction')
// class MockHsm extends buildClass('mockhsm', {
//   listPath: '/mockhsm/list-keys'
// }) {
//   create(context) {
//     let body = Object.assign({ client_token: uuid.v4() }, this)
//     return context.client.request('/mockhsm/create-key', body)
//       .then(data => new MockHsm(data))
//   }
//
//   // TODO: handle batch errors
//   static sign(transactions, xpubs, context) {
//     return context.client.request('/mockhsm/sign-transaction', {
//       transactions: transactions,
//       xpubs: xpubs
//     }).then(data => data.map((item) => new Transaction(item)))
//   }
// }

module.exports = (client) => {
  return {
    keys: {
      create: (params = {}) => {
        let body = Object.assign({ client_token: uuid.v4() }, params)
        return client.request('/mockhsm/create-key', body)
          .then(data => data)
      }
    }
  }
}
