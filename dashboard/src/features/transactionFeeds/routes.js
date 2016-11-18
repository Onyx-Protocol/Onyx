import { List, New } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(
  store, 'transactionFeed', List, New, null, { path: 'transaction-feeds', name: 'Transaction feeds'}
)
