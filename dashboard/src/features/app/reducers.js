import React from 'react'
import { Link } from 'react-router'
import { combineReducers } from 'redux'
import uuid from 'uuid'

const flash = (message, title, type) => ({ message, title, type, displayed: false })
const success = (message, title) => flash(message, title, 'success')
const error = (message, title) => flash(message, title, 'danger')

export const flashMessages = (state = new Map(), action) => {
  if (action.type == 'CREATED_ACCOUNT') {
    return new Map(state).set(uuid.v4(), success(<p>
      Created account. <Link to='accounts/create'>Create another?</Link>
    </p>))
  } else if (action.type == 'CREATED_ASSET') {
    return new Map(state).set(uuid.v4(), success(<p>
      Created asset. <Link to='assets/create'>Create another?</Link>
    </p>))
  } else if (action.type == 'CREATED_TRANSACTION') {
    return new Map(state).set(uuid.v4(), success(<p>
      Submitted transaction. <Link to='transactions/create'>Create another?</Link>
    </p>))
  } else if (action.type == 'CREATED_MOCKHSM') {
    return new Map(state).set(uuid.v4(), success('Created key'))
  } else if (action.type == 'CREATED_TRANSACTIONFEED') {
    return new Map(state).set(uuid.v4(), success('Created transaction feed'))
  } else if (['CREATED_CLIENT_ACCESS_TOKEN',
              'CREATED_NETWORK_ACCESS_TOKEN'].includes(action.type)) {
    const object = action.param
    return new Map(state).set(uuid.v4(), success(object.token, 'Created Access Token:'))
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
  } else if (action.type == 'USER_LOG_IN') {
    return new Map()
  }

  return state
}

export const modal = (state = { isShowing: false }, action) => {
  if      (action.type == 'SHOW_MODAL') return { isShowing: true, ...action.payload }
  else if (action.type == 'HIDE_MODAL') return { isShowing: false }
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
  flashMessages,
  modal,
  dropdownState,
})
