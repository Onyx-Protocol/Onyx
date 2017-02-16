import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

export default {
  clientAccessToken: combineReducers({
    items: reducers.itemsReducer('clientAccessToken'),
    queries: reducers.queriesReducer('clientAccessToken'),
  }),
  networkAccessToken: combineReducers({
    items: reducers.itemsReducer('networkAccessToken'),
    queries: reducers.queriesReducer('networkAccessToken'),
  }),
}
