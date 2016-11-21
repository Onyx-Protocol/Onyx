import buildClass from './buildClass'
import uuid from 'uuid'

export default class TransactionFeed extends buildClass('transaction-feed') {
  create(context) {
    let body = Object.assign({ client_token: uuid.v4() }, this)
    return context.client.request('/create-transaction-feed', body)
      .then(data => new this.constructor(data))
  }

  static delete(context, id) {
    return context.client.request('/delete-transaction-feed', {id})
  }
}
