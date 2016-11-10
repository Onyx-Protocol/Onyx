import buildClass from './buildClass'
import uuid from 'uuid'

export default class AccessToken extends buildClass('access-token') {
  create(context) {
    let body = Object.assign({ client_token: uuid.v4() }, this)
    return context.client.request('/create-access-token', body)
      .then(data => new this.constructor(data))
  }

  static delete(context, id) {
    return context.client.request('/delete-access-token', {id})
  }
}
