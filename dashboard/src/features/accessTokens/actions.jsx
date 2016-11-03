import {
  baseCreateActions,
  baseListActions
} from 'features/shared/actions'

import React from 'react'
import CreateModal from './components/CreateModal'

const makeCreateModal = token => {
  return <CreateModal token={token.token} />
}

let actions = {
  client_access_token: {
    ...baseListActions('client_access_token', {
      listPath: '/access_tokens/client',
      className: 'AccessToken',
      requiredParams: { type: 'client'},
    }),
    ...baseCreateActions('client_access_token', {
      listPath: '/access_tokens/client',
      className: 'AccessToken',
      createModal: makeCreateModal,
    }),
  },
  network_access_token: {
    ...baseListActions('network_access_token', {
      listPath: '/access_tokens/network',
      className: 'AccessToken',
      requiredParams: { type: 'network'},
    }),
    ...baseCreateActions('network_access_token', {
      listPath: '/access_tokens/network',
      className: 'AccessToken',
      createModal: makeCreateModal,
    }),
  }
}

export default actions
