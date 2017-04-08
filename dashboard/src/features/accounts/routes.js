import { List, New, AccountShow, AccountUpdate } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'account', List, New, AccountShow, AccountUpdate)
