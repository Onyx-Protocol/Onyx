import { List, New, TransactionDetail, GeneratedTxHex } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => {
  return makeRoutes(
    store,
    'transaction',
    List,
    New,
    TransactionDetail,
    null,
    {
      childRoutes: [
        {
          path: 'generated/:id',
          component: GeneratedTxHex,
        },
      ]
    }
  )
}
