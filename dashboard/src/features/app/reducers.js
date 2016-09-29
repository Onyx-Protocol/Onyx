import { combineReducers } from 'redux'
import uuid from 'uuid'

const flash = (message, type) => ({ message, type, displayed: false })
const success = (message) => flash(message, 'success')
const error = (message) => flash(message, 'danger')

export const flashMessages = (state = new Map(), action) => {
  if (action.type == 'CREATED_ACCOUNT') {
    return new Map(state).set(uuid.v4(), success('Created account'))
  } else if (action.type == 'CREATED_ASSET') {
    return new Map(state).set(uuid.v4(), success('Created asset'))
  } else if (action.type == 'CREATED_TRANSACTION') {
    return new Map(state).set(uuid.v4(), success('Created transaction'))
  } else if (action.type == 'CREATE_MOCKHSM') {
    return new Map(state).set(uuid.v4(), success('Created key'))
  } else if (action.type == 'ERROR') {
    return new Map(state).set(uuid.v4(), error(action.payload.message))
  } else if (action.type == 'DISPLAYED_FLASH') {
    const existing = state.get(action.param)
    if (existing && !existing.displayed) {
      const newState = new Map(state)
      existing.displayed = true
      newState.set(action.param, existing)
      return newState
    }
    return state
  } else if (action.type == '@@router/LOCATION_CHANGE') {
    if (action.payload.state && action.payload.state.preserveFlash) {
      return state
    } else {
      state.forEach((item, key) => {
        if (item.displayed) {
          state.delete(key)
        }
      })
      return new Map(state)
    }
  } else if (action.type == 'DISMISS_FLASH') {
    state.delete(action.param)
    return new Map(state)
  }

  return state
}

export const dropdownState = (state = '', action) => {
  if (action.type == 'TOGGLE_DROPDOWN') {
    return state === '' ? 'open' : ''
  } else if (action.type == 'CLOSE_DROPDOWN') {
    return ''
  }

  return state
}

export default combineReducers({
  dropdownState,
  flashMessages
})
