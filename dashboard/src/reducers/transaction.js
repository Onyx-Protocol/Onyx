import { pagesActions, currentPageActions, currentQueryActions } from './base'
import { combineReducers } from 'redux'

const type = "transaction"

export default combineReducers({
  pages: pagesActions(type),
  currentPage: currentPageActions(type),
  currentQuery: currentQueryActions(type)
})
