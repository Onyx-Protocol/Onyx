import {
  baseFormActions,
  baseListActions
} from 'features/shared/actions'

import { actions as coreActions } from 'features/core'
import chain from 'chain'
import { context } from 'utility/environment'

const setRequired = (type, value) => {
  return (dispatch) => chain.Core.updateConfiguration(context(), {
    [`require_${type}s`]: value
  }).then(() => dispatch(coreActions.fetchCoreInfo()))
    .then(() => dispatch({type: 'UPDATED_CONFIGURATION'}))
    .catch(err => dispatch({type: 'ERROR', payload: err}))
}

let actions = {
  client_access_token: {
    ...baseListActions('client_access_token', {
      listPath: '/access_tokens/client',
      className: 'AccessToken',
      requiredParams: { type: 'client'},
    }),
    ...baseFormActions('client_access_token', {
      listPath: '/access_tokens/client',
      className: 'AccessToken',
    }),
    enable: setRequired('client_access_token', true),
    disable: setRequired('client_access_token', false),
  },
  network_access_token: {
    ...baseListActions('network_access_token', {
      listPath: '/access_tokens/network',
      className: 'AccessToken',
      requiredParams: { type: 'network'},
    }),
    ...baseFormActions('network_access_token', {
      listPath: '/access_tokens/network',
      className: 'AccessToken',
    }),
    enable: setRequired('network_access_token', true),
    disable: setRequired('network_access_token', false),
  }
}

export default actions
