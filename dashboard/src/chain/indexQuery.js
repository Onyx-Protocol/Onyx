import buildClass from './buildClass'
import uuid from 'uuid'

export default class Index extends buildClass('index', {
  listPath: 'list-indexes'
}) {
  create(context) {
    let body = Object.assign({ client_token: uuid.v4() }, this)
    return context.client.request('/create-index', body)
      .then(data => new Index(data))
  }
}
