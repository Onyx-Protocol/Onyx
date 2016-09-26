import {
  currentPageReducer,
  currentCursorReducer,
  currentQueryReducer,
  currentQueryTimeReducer
} from './base'

import { combineReducers } from 'redux'

const type = 'balance'

const itemsReducer = (state = {}, action) => {
  // Since balance does not support pagination,
  // receiving a new balance page completely replaces
  // the old one
  if (action.type == 'APPEND_BALANCE_PAGE') {
    const newState = {}
    action.param.items.forEach((item, index) => {
      item.id = `balance-${index}`
      newState[index] = item
    })
    return newState
  }
  return state
}

const currentListReducer = (state = [], action) => {
  if (action.type == 'UPDATE_BALANCE_QUERY') {
    return []
  } else if (action.type == 'APPEND_BALANCE_PAGE') {
    return action.param.items.map((item, index) => index)
  }
  return state
}

const sumByReducers = (state = '', action) => {
  if (action.type == 'UPDATE_BALANCE_QUERY') {
    if (action.param && action.param.sumBy) {
      return action.param.sumBy
    }
    return ''
  }

  return state
}

export default combineReducers({
  items: itemsReducer,
  listView: combineReducers({
    itemIds: currentListReducer,
    cursor: currentCursorReducer(type),
    pageIndex: currentPageReducer(type),
    query: currentQueryReducer(type),
    queryTime: currentQueryTimeReducer(type),
    sumBy: sumByReducers,
  })
})
