import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'unspent'
const idFunc = item => `${item.transaction_id}-${item.position}`

export default combineReducers({
  items: reducers.itemsReducer(type, idFunc),
  queries: reducers.queriesReducer(type, idFunc)
})
