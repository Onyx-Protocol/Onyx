import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'access_token'

export default {
  client_access_token: combineReducers({
    items: reducers.itemsReducer('client_' + type),
    listView: reducers.listViewReducer('client_' + type),
  }),
  network_access_token: combineReducers({
    items: reducers.itemsReducer('network_' + type),
    listView: reducers.listViewReducer('network_' + type),
  }),
}
