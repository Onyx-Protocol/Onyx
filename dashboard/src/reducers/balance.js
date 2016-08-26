import { pagesActions, currentPageActions, currentQueryActions } from './base'
import { combineReducers } from 'redux'
import actions from '../actions'

const type = "balance"

const sumByReducers = (state = "", action) => {
  if (action.type == actions[type].updateQuery.type) {
    if (action.param && action.param.sumBy) {
      return action.param.sumBy
    }
    return ""
  }

  return state
}

export default combineReducers({
  pages: pagesActions(type),
  currentPage: currentPageActions(type),
  currentQuery: currentQueryActions(type),
  sumBy: sumByReducers
})
