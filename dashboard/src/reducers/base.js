import { combineReducers } from 'redux'

const defaultIdFunc = (item) => item.id

export const itemsReducer = (type, idFunc = defaultIdFunc) => (state = {}, action) => {
  if (action.type == `APPEND_${type.toUpperCase()}_PAGE`) {
    const newObjects = {}
    action.param.items.forEach(item => {
      if (!item.id) { item.id = idFunc(item) }
      newObjects[idFunc(item)] = item
    })
    return {...state, ...newObjects}
  }
  return state
}

export const currentListReducer = (type, idFunc = defaultIdFunc) => (state = [], action) => {
  if ([`CREATED_${type.toUpperCase()}`,
       `UPDATE_${type.toUpperCase()}_QUERY`].includes(action.type)) {
    return []
  } else if (action.type == `APPEND_${type.toUpperCase()}_PAGE`) {
    const newItemIds = [...state, ...action.param.items.map(item => idFunc(item))]
    return [...new Set(newItemIds)]
  }
  return state
}

export const currentCursorReducer = (type) => (state = {}, action) => {
  if ([`CREATED_${type.toUpperCase()}`,
       `UPDATE_${type.toUpperCase()}_QUERY`].includes(action.type)) {
    return {}
  } else if (action.type == `APPEND_${type.toUpperCase()}_PAGE`) {
    return action.param
  }
  return state
}

export const currentPageReducer = (type) => (state = 0, action) => {
  if ([`CREATED_${type.toUpperCase()}`,
       `UPDATE_${type.toUpperCase()}_QUERY`].includes(action.type)) {
    return 0
  } else if (action.type == `INCREMENT_${type.toUpperCase()}_PAGE`) {
    return state + 1
  } else if (action.type == `DECREMENT_${type.toUpperCase()}_PAGE`) {
    return Math.max(state - 1, 0)
  }

  return state
}

export const currentQueryReducer = (type) => (state = '', action) => {
  if (action.type == `UPDATE_${type.toUpperCase()}_QUERY`) {
    if (action.param && action.param.query) {
      return action.param.query
    } else if (typeof action.param === 'string') {
      return action.param
    }

    return ''
  } else if (action.type == `CREATED_${type.toUpperCase()}`) {
    return ''
  }

  return state
}

export const listViewReducer = (type, idFunc = defaultIdFunc) => combineReducers({
  itemIds: currentListReducer(type, idFunc),
  cursor: currentCursorReducer(type),
  pageIndex: currentPageReducer(type),
  query: currentQueryReducer(type)
})
