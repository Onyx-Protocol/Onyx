import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'transactionConsumer'

export default combineReducers({
  items: reducers.itemsReducer(type),
  listView: reducers.listViewReducer(type),
})
