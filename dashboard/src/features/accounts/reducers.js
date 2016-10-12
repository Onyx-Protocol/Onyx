import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'account'

export default combineReducers({
  items: reducers.itemsReducer(type),
  queries: reducers.queriesReducer(type),
  autocompleteIsLoaded: reducers.autocompleteIsLoadedReducer(type),
})
