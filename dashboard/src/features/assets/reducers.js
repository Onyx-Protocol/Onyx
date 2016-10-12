import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'asset'

export default combineReducers({
  items: reducers.itemsReducer(type),
  queries: reducers.queriesReducer(type),
  autocompleteIsLoaded: reducers.autocompleteIsLoadedReducer(type),
})
