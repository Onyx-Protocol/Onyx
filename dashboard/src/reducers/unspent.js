import {
  itemsReducer,
  listViewReducer
} from './base'
import { combineReducers } from 'redux'

const type = 'unspent'
const idFunc = item => `${item.transaction_id}-${item.position}`

export default combineReducers({
  items: itemsReducer(type, idFunc),
  listView: listViewReducer(type, idFunc)
})
