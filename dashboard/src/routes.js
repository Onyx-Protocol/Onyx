import { Container } from 'features/app/components'
import { NotFound } from 'features/shared/components'
import { routes as accessTokens } from 'features/accessTokens'
import { routes as accounts } from 'features/accounts'
import { routes as assets } from 'features/assets'
import { routes as balances } from 'features/balances'
import { routes as configuration } from 'features/configuration'
import { routes as core } from 'features/core'
import { routes as transactions } from 'features/transactions'
import { routes as unspents } from 'features/unspents'
import { routes as mockhsm } from 'features/mockhsm'

export default ({
  path: '/',
  component: Container,
  childRoutes: [
    accessTokens,
    accounts,
    assets,
    balances,
    configuration,
    core,
    transactions,
    unspents,
    mockhsm,
    {
      path: '*',
      component: NotFound
    }
  ]
})
