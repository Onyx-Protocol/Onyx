import { pagesActions, currentPageActions } from './base'
import { combineReducers } from 'redux'

const type = "mockhsm"

export default combineReducers({
  pages: pagesActions(type),
  currentPage: currentPageActions(type),
})
