import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'unspent'
const idFunc = item => `${item.id}`

export default combineReducers({
  items: reducers.itemsReducer(type, idFunc),
  queries: reducers.queriesReducer(type, idFunc)
})
