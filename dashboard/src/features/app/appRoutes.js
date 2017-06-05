import accessControl from 'features/accessControl/routes'
import { routes as accounts } from 'features/accounts'
import { routes as assets } from 'features/assets'
import { routes as balances } from 'features/balances'
import { routes as core } from 'features/core'
import { routes as transactions } from 'features/transactions'
import { routes as transactionFeeds } from 'features/transactionFeeds'
import { routes as unspents } from 'features/unspents'
import { routes as mockhsm } from 'features/mockhsm'
import { NotFound } from 'features/shared/components'

import Layout from './components/Layout/Layout'

export default (store) => ({
  useForBreadcrumbs: true, // key for app functions inspecting routes
  component: Layout,
  childRoutes: [
    transactions(store),
    accessControl(store),
    accounts(store),
    assets(store),
    balances(store),
    core,
    transactionFeeds(store),
    unspents(store),
    mockhsm(store),
    {
      path: '*',
      component: NotFound
    }
  ]
})
