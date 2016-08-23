import { pagesActions, currentPageActions, currentQueryActions } from './base'
import { combineReducers } from 'redux'

const type = "balance"

const queryReducers = (state = "", action) => {
  let newState = currentQueryActions(type)(state, action)

  if (newState == "") {
    return "asset_id=$1 AND asset_alias=$2"
  } else {
    return newState
  }
}

export default combineReducers({
  pages: pagesActions(type),
  currentPage: currentPageActions(type),
  currentQuery: queryReducers
})
