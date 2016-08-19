import { pagesActions, currentPageActions } from './base'
import { combineReducers } from 'redux'

const type = "index"

export default combineReducers({
  pages: pagesActions(type),
  currentPage: currentPageActions(type)
})
