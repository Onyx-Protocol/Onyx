import { Container } from 'features/app/components'
import { RoutingContainer } from 'features/shared/components'
import UnspentList from 'containers/Unspent/List'
import BalanceList from 'containers/Balance/List'
import MockHsmList from 'containers/MockHsm/List'
import NewKey from 'containers/MockHsm/New'
import { NotFound } from 'features/shared/components'

import { routes as accessTokens } from 'features/accessTokens'
import { routes as accounts } from 'features/accounts'
import { routes as assets } from 'features/assets'
import { routes as configuration } from 'features/configuration'
import { routes as core } from 'features/core'
import { routes as transactions } from 'features/transactions'

export default ({
  path: '/',
  component: Container,
  childRoutes: [
    assets,
    accounts,
    transactions,
    accessTokens,
    configuration,
    core,
    {
      path: 'unspents',
      component: RoutingContainer,
      indexRoute: { component: UnspentList },
    },
    {
      path: 'balances',
      component: RoutingContainer,
      indexRoute: { component: BalanceList },
    },
    {
      path: 'mockhsms',
      component: RoutingContainer,
      indexRoute: { component: MockHsmList },
      childRoutes: [{ path: 'create', component: NewKey }]
    },
    {
      path: '*',
      component: NotFound
    }
  ]
})
