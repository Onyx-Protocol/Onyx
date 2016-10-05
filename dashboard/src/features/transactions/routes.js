import Section from 'containers/SectionContainer'
import TransactionList from 'containers/Transactions/List'
import NewTransaction from 'containers/Transactions/New'
import { Show, GeneratedTxHex } from './components'

export default {
  path: 'transactions',
  component: Section,
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
