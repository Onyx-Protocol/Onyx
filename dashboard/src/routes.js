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

import Main from 'features/app/components/Main/Main'


const makeRoutes = (store) => {
  return({
    ...container(store),
    childRoutes: [
      authn(store),
      configuration,
      {
        useForBreadcrumbs: true, // key for app functions inspecting routes
        component: Main,
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
      },
    ]
  })
}
export default makeRoutes
