import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'mockhsm'
const idFunc = item => item.xpub

export default combineReducers({
  items: reducers.itemsReducer(type, idFunc),
  listView: reducers.listViewReducer(type, idFunc),
  autocompleteIsLoaded: reducers.autocompleteIsLoadedReducer(type),
})
