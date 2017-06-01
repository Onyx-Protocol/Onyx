import { routes as container } from 'features/container'
import accessControl from 'features/accessControl/routes'
import { routes as accounts } from 'features/accounts'
import { routes as assets } from 'features/assets'
import { routes as authn } from 'features/authn'
import { routes as balances } from 'features/balances'
import { routes as configuration } from 'features/configuration'
import { routes as core } from 'features/core'
import { routes as transactions } from 'features/transactions'
import { routes as transactionFeeds } from 'features/transactionFeeds'
import { routes as unspents } from 'features/unspents'
import { routes as mockhsm } from 'features/mockhsm'
import { NotFound } from 'features/shared/components'

const makeRoutes = (store) => ({
  ...container(store),
  childRoutes: [
    accessControl(store),
    accounts(store),
    assets(store),
    authn(store),
    balances(store),
    configuration,
    core,
    transactions(store),
    transactionFeeds(store),
    unspents(store),
    mockhsm(store),
    {
      path: '*',
      component: NotFound
    }
  ]
})

export default makeRoutes
