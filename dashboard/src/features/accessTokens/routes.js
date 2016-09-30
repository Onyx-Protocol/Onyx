import Section from 'containers/SectionContainer'
import { ClientTokenList, NetworkTokenList } from './components/List'
import { NewClientToken, NewNetworkToken } from './components/New'

export default {
  path: 'access_tokens',
  indexRoute: {
    onEnter: ({ params }, replace) => replace('access_tokens/client')
  },
  childRoutes: [
    {
      path: 'client',
      component: Section,
      indexRoute: { component: ClientTokenList },
      childRoutes: [{
        path: 'create',
        component: NewClientToken
      }]
    },
    {
      path: 'network',
      component: Section,
      indexRoute: { component: NetworkTokenList },
      childRoutes: [{
        path: 'create',
        component: NewNetworkToken
      }]
    }
  ]
}
