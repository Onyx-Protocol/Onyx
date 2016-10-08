import { RoutingContainer } from 'features/shared/components'
import { Show, List, New, GeneratedTxHex } from './components'
import { routes as transactionFeeds } from 'features/transactionFeeds'

export default {
  path: 'transactions',
  component: RoutingContainer,
  indexRoute: { component: List },
  childRoutes: [
    transactionFeeds,
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
