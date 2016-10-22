import { List, New, Show, GeneratedTxHex } from './components'
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
        {
          path: 'generated/:id',
          component: GeneratedTxHex,
        },
      ]
    }
  )
}
