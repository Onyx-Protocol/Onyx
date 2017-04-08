import { List } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'balance', List, null, null, null, {
  defaultFilter: "is_local='yes'"
})
