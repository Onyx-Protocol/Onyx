import { RoutingContainer } from 'features/shared/components'
import TransactionList from 'containers/Transactions/List'
import NewTransaction from 'containers/Transactions/New'
import { Show, GeneratedTxHex } from './components'

export default {
  path: 'transactions',
  component: RoutingContainer,
  indexRoute: { component: TransactionList },
  childRoutes: [
    {
      path: 'create',
      component: NewTransaction
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
