import buildClass from './buildClass'
import uuid from 'uuid'

export default class TransactionConsumer extends buildClass('transaction-consumer') {
  create(context) {
    let body = Object.assign({ client_token: uuid.v4() }, this)
    return context.client.request('/create-transaction-consumer', body)
      .then(data => new this.constructor(data))
  }

  static delete(context, id) {
    return context.client.request('/delete-transaction-consumer', {id})
  }
}
