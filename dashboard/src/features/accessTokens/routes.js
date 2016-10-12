import { ClientTokenList, NetworkTokenList } from './components/List'
import { NewClientToken, NewNetworkToken } from './components/New'
import { makeRoutes } from 'features/shared'

export default (store) => {
  return {
    path: 'access_tokens',
    indexRoute: {
      onEnter: ({ params }, replace) => replace('access_tokens/client')
    },
    childRoutes: [
      makeRoutes(
        store,
        'client_access_token',
        ClientTokenList, NewClientToken, null,
        { path: 'client', skipFilter: true }
      ),
      makeRoutes(
        store,
        'network_access_token',
        NetworkTokenList, NewNetworkToken, null,
        { path: 'network', skipFilter: true }
      ),
    ]
  }
}
