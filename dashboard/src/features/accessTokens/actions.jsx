import {
  baseCreateActions,
  baseListActions
} from 'features/shared/actions'
import { chainClient } from 'utility/environment'

import React from 'react'
import CreateModal from './components/CreateModal'

const makeCreateModal = token => {
  return <CreateModal token={token.token} />
}

let actions = {
  clientAccessToken: {
    ...baseListActions('clientAccessToken', {
      listPath: '/access_tokens/client',
      className: 'AccessToken',
      requiredParams: { type: 'client'},
      clientApi: () => chainClient().accessTokens,
    }),
    ...baseCreateActions('clientAccessToken', {
      listPath: '/access_tokens/client',
      className: 'AccessToken',
      createModal: makeCreateModal,
      clientApi: () => chainClient().accessTokens,
    }),
  },
  networkAccessToken: {
    ...baseListActions('networkAccessToken', {
      listPath: '/access_tokens/network',
      className: 'AccessToken',
      requiredParams: { type: 'network'},
      clientApi: () => chainClient().accessTokens,
    }),
    ...baseCreateActions('networkAccesToken', {
      listPath: '/access_tokens/network',
      className: 'AccessToken',
      createModal: makeCreateModal,
      clientApi: () => chainClient().accessTokens,
    }),
  }
}

export default actions
