import { List, New } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'mockhsm', List, New, null, { skipFilter: true, name: 'MockHSM keys' })
