import { combineReducers } from 'redux'

const connected = (state = false, action) => {
  if      (action.type == 'CONNECTED_TO_CORE' ||
           action.type == 'UPDATE_CORE_INFO') return true
  else if (action.type == 'CORE_DISCONNECT')  return false

  return state
}

const authenticationRequired = (state = false, action) => {
  if (action.type == 'AUTHENTICATION_REQUIRED') return true

  return state
}

const authenticated = (state = false, action) => {
  if      (action.type == 'AUTHENTICATION_INVALID') return false
  else if (action.type == 'UPDATE_CORE_INFO') return true

  return state
}

const clientToken = (state = '', action) => {
  if      (action.type == 'SET_CLIENT_TOKEN') return action.token
  else if (action.type == 'ERROR' &&
           action.payload.status == 401)      return ''

  return state
}

const authenticationReady = (state = false, action) => {
  if (action.type == 'AUTHENTICATION_READY') return true

  return state
}

export default combineReducers({
  connected,
  authenticationRequired,
  clientToken,
  authenticated,
  authenticationReady,
})
