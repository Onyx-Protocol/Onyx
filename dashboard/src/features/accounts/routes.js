import { List, New, Show } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'account', List, New, Show)
