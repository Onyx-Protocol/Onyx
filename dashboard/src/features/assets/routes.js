import { RoutingContainer } from 'features/shared/components'
import List from '../../containers/Assets/List'
import New from '../../containers/Assets/New'
import Show from './components/Show'

export default {
  path: 'assets',
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
