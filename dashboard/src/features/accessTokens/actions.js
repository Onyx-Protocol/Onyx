import generateListActions from 'actions/listActions'
import generateFormActions from 'actions/formActions'

const type = 'access_token'

let actions = {
  client_access_token: {
    ...generateListActions('client_' + type, {
      className: 'AccessToken',
      requiredParams: { type: 'client'},
    }),
    ...generateFormActions('client_' + type, {
      listPath: '/access_tokens/client',
      className: 'AccessToken',
    }),
  },
  network_access_token: {
    ...generateListActions('network_' + type, {
      className: 'AccessToken',
      requiredParams: { type: 'network'},
    }),
    ...generateFormActions('network_' + type, {
      listPath: '/access_tokens/network',
      className: 'AccessToken',
    }),
  }
}

export default actions
