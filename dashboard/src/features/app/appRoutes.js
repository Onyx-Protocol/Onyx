import accessControlRoutes from 'features/accessControl/routes'
import accountRoutes from 'features/accounts/routes'
import assetRoutes from 'features/assets/routes'
import balanceRoutes from 'features/balances/routes'
import coreRoutes from 'features/core/routes'
import transactionRoutes from 'features/transactions/routes'
import transactionFeedRoutes from 'features/transactionFeeds/routes'
import unspentRoutes from 'features/unspents/routes'
import mockhsmRoutes from 'features/mockhsm/routes'
import NotFound from 'features/shared/components/NotFound'

import Layout from './components/Layout/Layout'

export default (store) => ({
  useForBreadcrumbs: true, // key for app functions inspecting routes
  component: Layout,
  childRoutes: [
    transactionRoutes(store),
    accessControlRoutes(store),
    accountRoutes(store),
    assetRoutes(store),
    balanceRoutes(store),
    coreRoutes,
    transactionFeedRoutes(store),
    unspentRoutes(store),
    mockhsmRoutes(store),
    {
      path: '*',
      component: NotFound
    }
  ]
})
