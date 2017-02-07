import { List, New, AccountShow } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'account', List, New, AccountShow)
