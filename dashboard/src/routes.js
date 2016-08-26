import Navigation from './components/Navigation/Navigation'
import Section from './containers/SectionContainer'

import TransactionList from './containers/Transactions/List'
import NewTransaction from './containers/Transactions/New'

import UnspentList from './containers/Unspent/List'
import BalanceList from './containers/Balance/List'

import AccountList from './containers/Accounts/List'
import NewAccount from './containers/Accounts/New'

import AssetList from './containers/Assets/List'
import NewAsset from './containers/Assets/New'

import IndexList from './containers/Indexes/List'
import NewIndex from './containers/Indexes/New'

import MockHsmList from './containers/MockHsm/List'
import NewKey from './containers/MockHsm/New'

import CoreSettings from './components/CoreSettings/Index'

import NotFound from './components/NotFound'

export default ({
  path: '/',
  component: Navigation,
  indexRoute: { onEnter: (nextState, replace) => replace('/transactions') },
  childRoutes: [
    {
      path: 'transactions',
      component: Section,
      indexRoute: { component: TransactionList },
      childRoutes: [{ path: 'create', component: NewTransaction }]
    },
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
      path: 'accounts',
      component: Section,
      indexRoute: { component: AccountList },
      childRoutes: [{ path: 'create', component: NewAccount }]
    },
    {
      path: 'assets',
      component: Section,
      indexRoute: { component: AssetList },
      childRoutes: [{ path: 'create', component: NewAsset }]
    },
    {
      path: 'indexes',
      component: Section,
      indexRoute: { component: IndexList },
      childRoutes: [{ path: 'create', component: NewIndex }]
    },
    {
      path: 'mockhsms',
      component: Section,
      indexRoute: { component: MockHsmList },
      childRoutes: [{ path: 'create', component: NewKey }]
    },
    {
      path: 'core-settings',
      component: CoreSettings,
    },
    {
      path: '*',
      component: NotFound
    }
  ]
})
