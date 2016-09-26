import {
  itemsReducer,
  listViewReducer
} from '../../reducers/base'
import { combineReducers } from 'redux'

const type = 'transaction'

export default combineReducers({
  items: itemsReducer(type),
  listView: listViewReducer(type)
})
