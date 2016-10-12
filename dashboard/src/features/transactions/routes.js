import { List, New, Show, GeneratedTxHex } from './components'
import { routes as transactionFeeds } from 'features/transactionFeeds'
import { makeRoutes } from 'features/shared'

export default (store) => {
  return makeRoutes(
    store,
    'transaction',
    List,
    New,
    Show,
    {
      childRoutes: [
        transactionFeeds(store),
        {
          path: 'generated/:id',
          component: GeneratedTxHex,
        },
      ]
    }
  )
}
