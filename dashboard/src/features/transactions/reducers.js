import {
  itemsReducer,
  listViewReducer
} from '../../reducers/base'
import { combineReducers } from 'redux'

const type = 'transaction'
const maxGeneratedHistory = 50

export default combineReducers({
  items: itemsReducer(type),
  listView: listViewReducer(type),
  generated: (state = [], action) => {
    if (action.type == 'GENERATED_TX_HEX') {
      return [action.generated, ...state].slice(0, maxGeneratedHistory)
    }
    return state
  },
})
