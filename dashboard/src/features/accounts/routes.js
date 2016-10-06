import { RoutingContainer } from 'features/shared/components'
import List from 'containers/Accounts/List'
import New from 'containers/Accounts/New'
import Show from './components/Show'

export default {
  path: 'accounts',
  component: RoutingContainer,
  indexRoute: { component: List },
  childRoutes: [
    {
      path: 'create',
      component: New
    },
    {
      path: ':id',
      component: Show
    }
  ]
}
