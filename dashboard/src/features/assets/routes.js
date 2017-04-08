import { List, New, AssetShow, AssetUpdate } from './components'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'asset', List, New, AssetShow, AssetUpdate)
