import {
  itemsReducer,
  listViewReducer,
  autocompleteIsLoadedReducer,
} from './base'
import { combineReducers } from 'redux'

const type = 'mockhsm'
const idFunc = item => item.xpub

export default combineReducers({
  items: itemsReducer(type, idFunc),
  listView: listViewReducer(type, idFunc),
  autocompleteIsLoaded: autocompleteIsLoadedReducer(type),
})
