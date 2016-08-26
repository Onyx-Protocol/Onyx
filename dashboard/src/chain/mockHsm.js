import buildClass from './buildClass'
import uuid from 'uuid'

export default class MockHsm extends buildClass('mockhsm', {
  listPath: "/mockhsm/list-keys"
}) {
  create(context) {
    let body = Object.assign({ client_token: uuid.v4() }, this)
    return context.client.request('/mockhsm/create-key', body)
      .then(data => new MockHsm(data))
  }

  static sign(templates, context) {
    return context.client.request('/mockhsm/sign-transaction-template', templates)
      .then(data => data)
  }
}
