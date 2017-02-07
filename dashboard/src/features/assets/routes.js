import { List, New, AssetShow } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'asset', List, New, AssetShow)
