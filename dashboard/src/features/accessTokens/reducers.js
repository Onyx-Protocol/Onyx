import {
  itemsReducer,
  listViewReducer,
} from '../../reducers/base'
import { combineReducers } from 'redux'

const type = 'access_token'

export default {
  client_access_token: combineReducers({
    items: itemsReducer('client_' + type),
    listView: listViewReducer('client_' + type),
  }),
  network_access_token: combineReducers({
    items: itemsReducer('network_' + type),
    listView: listViewReducer('network_' + type),
  }),
}
