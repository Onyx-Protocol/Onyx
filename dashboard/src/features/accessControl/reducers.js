import { combineReducers } from 'redux'
import { reducers } from 'features/shared'

const type = 'accessControl'

const idFunc = (item) => `${JSON.stringify(item.guardData)}-${item.policy}`

export default combineReducers({
  items: reducers.itemsReducer(type, idFunc),
  queries: reducers.queriesReducer(type, idFunc),
})
