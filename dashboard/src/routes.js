import AppContainer from './containers/AppContainer'
import Section from './containers/SectionContainer'
import UnspentList from './containers/Unspent/List'
import BalanceList from './containers/Balance/List'
import MockHsmList from './containers/MockHsm/List'
import NewKey from './containers/MockHsm/New'
import CoreIndex from './containers/Core'
import ConfigIndex from './containers/Config'
import NotFound from './components/NotFound'

import { routes as transactions } from './features/transactions'
import { routes as assets } from './features/assets'
import { routes as accounts } from './features/accounts'

export default ({
  path: '/',
  component: AppContainer,
  childRoutes: [
    assets,
    accounts,
    transactions,
    {
      path: 'unspents',
      component: Section,
      indexRoute: { component: UnspentList },
    },
    {
      path: 'balances',
      component: Section,
      indexRoute: { component: BalanceList },
    },
    {
      path: 'mockhsms',
      component: Section,
      indexRoute: { component: MockHsmList },
      childRoutes: [{ path: 'create', component: NewKey }]
    },
    {
      path: 'core',
      component: Section,
      indexRoute: { component: CoreIndex }
    },
    {
      path: 'configuration',
      component: Section,
      indexRoute: { component: ConfigIndex }
    },
    {
      path: '*',
      component: NotFound
    }
  ]
})
