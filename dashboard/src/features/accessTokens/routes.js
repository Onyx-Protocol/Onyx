import { ClientTokenList, NetworkTokenList } from './components/List'
import { NewClientToken, NewNetworkToken } from './components/New'
import { makeRoutes } from 'features/shared'

export default (store) => {
  return {
    path: 'access_tokens',
    skipBreadcrumb: true,
    indexRoute: {
      onEnter: ({ params }, replace) => replace('access_tokens/client')
    },
    childRoutes: [
      makeRoutes(
        store,
        'clientAccessToken',
        ClientTokenList, NewClientToken, null,
        { path: 'client', skipFilter: true, name: 'Client access tokens' }
      ),
      makeRoutes(
        store,
        'networkAccessToken',
        NetworkTokenList, NewNetworkToken, null,
        { path: 'network', skipFilter: true, name: 'Network access tokens' }
      ),
    ]
  }
}
