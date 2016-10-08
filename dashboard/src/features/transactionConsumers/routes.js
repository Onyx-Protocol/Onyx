import { RoutingContainer } from 'features/shared/components'
import { List, New } from './components'

export default {
  path: 'consumers',
  component: RoutingContainer,
  indexRoute: { component: List },
  childRoutes: [
    {
      path: 'create',
      component: New
    },
  ]
}
