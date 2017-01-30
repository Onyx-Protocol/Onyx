import { combineReducers } from 'redux'
import moment from 'moment'
import uniq from 'lodash/uniq'

const defaultIdFunc = (item) => item.id

export const itemsReducer = (type, idFunc = defaultIdFunc) => (state = {}, action) => {
  if (action.type == `RECEIVED_${type.toUpperCase()}_ITEMS`) {
    const newObjects = {}
    action.param.items.forEach(item => {
      if (!item.id) { item.id = idFunc(item) }
      newObjects[idFunc(item)] = item
    })
    return {...state, ...newObjects}
  } else if (action.type == `DELETE_${type.toUpperCase()}`) {
    delete state[action.id]
    return {...state}
  }
  return state
}

export const queryItemsReducer = (type, idFunc = defaultIdFunc) => (state = [], action) => {
  if (action.type == `APPEND_${type.toUpperCase()}_PAGE`) {
    let newItemIds = action.param.items.map((item, index) => idFunc(item, index))

    if (action.refresh) return newItemIds
    else {
      return uniq([...state, ...newItemIds])
    }
  } else if (action.type == `DELETE_${type.toUpperCase()}`) {
    const index = state.indexOf(action.id)
    if (index >= 0) {
      state.splice(index, 1)
      return [...state]
    }
  }
  return state
}

export const queryCursorReducer = (type) => (state = {}, action) => {
  if (action.type == `APPEND_${type.toUpperCase()}_PAGE`) {
    return action.param
  }
  return state
}

export const queryTimeReducer = (type) => (state = '', action) => {
  if (action.type == `APPEND_${type.toUpperCase()}_PAGE`) {
    return moment().format('h:mm:ss a')
  }
  return state
}

export const autocompleteIsLoadedReducer = (type) => (state = false, action) => {
  if (action.type == `DID_LOAD_${type.toUpperCase()}_AUTOCOMPLETE`) {
    return true
  }

  return state
}

export const listViewReducer = (type, idFunc = defaultIdFunc) => combineReducers({
  itemIds: queryItemsReducer(type, idFunc),
  cursor: queryCursorReducer(type),
  queryTime: queryTimeReducer(type)
})

export const queriesReducer = (type, idFunc = defaultIdFunc) => (state = {}, action) => {
  if (action.type == `APPEND_${type.toUpperCase()}_PAGE`) {
    const query = action.param.next.filter || ''
    const list = state[query] || {}

    return {
      ...state,
      [query]: listViewReducer(type, idFunc)(list, action)
    }
  }

  return state
}
