import { reducers } from 'features/shared'
import { combineReducers } from 'redux'

const type = 'transaction'
const maxGeneratedHistory = 50

export default combineReducers({
  items: reducers.itemsReducer(type),
  queries: reducers.queriesReducer(type),
  generated: (state = [], action) => {
    if (action.type == 'GENERATED_TX_HEX') {
      return [action.generated, ...state].slice(0, maxGeneratedHistory)
    }
    return state
  },
})
