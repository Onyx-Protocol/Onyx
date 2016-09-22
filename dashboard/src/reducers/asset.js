import {
  itemsReducer,
  listViewReducer,
} from './base'
import { combineReducers } from 'redux'

const type = 'asset'

export default combineReducers({
  items: itemsReducer(type),
  listView: listViewReducer(type)
})
