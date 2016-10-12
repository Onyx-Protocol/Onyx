import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'access_token'

export default {
  client_access_token: combineReducers({
    items: reducers.itemsReducer('client_' + type),
    queries: reducers.queriesReducer('client_' + type),
  }),
  network_access_token: combineReducers({
    items: reducers.itemsReducer('network_' + type),
    queries: reducers.queriesReducer('network_' + type),
  }),
}
