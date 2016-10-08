import { RoutingContainer } from 'features/shared/components'
import { Show, List, New, GeneratedTxHex } from './components'
import { routes as transactionConsumers } from 'features/transactionConsumers'

export default {
  path: 'transactions',
  component: RoutingContainer,
  indexRoute: { component: List },
  childRoutes: [
    transactionConsumers,
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
