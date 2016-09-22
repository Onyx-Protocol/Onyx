import {
  itemsReducer,
  listViewReducer,
} from './base'
import { combineReducers } from 'redux'

const type = 'account'

export default combineReducers({
  items: itemsReducer(type),
  listView: listViewReducer(type)
})
