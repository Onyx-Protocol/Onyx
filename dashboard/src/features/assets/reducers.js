import {
  itemsReducer,
  listViewReducer,
  autocompleteIsLoadedReducer,
} from '../../reducers/base'
import { combineReducers } from 'redux'

const type = 'asset'

export default combineReducers({
  items: itemsReducer(type),
  listView: listViewReducer(type),
  autocompleteIsLoaded: autocompleteIsLoadedReducer(type),
})
