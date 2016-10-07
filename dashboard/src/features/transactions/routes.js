import { RoutingContainer } from 'features/shared/components'
import { Show, List, New, GeneratedTxHex } from './components'

export default {
  path: 'transactions',
  component: RoutingContainer,
  indexRoute: { component: List },
  childRoutes: [
    {
      path: 'create',
      component: New
    },
    {
      path: 'generated/:id',
      component: GeneratedTxHex,
    },
    {
      path: ':id',
      component: Show
    },
  ]
}
