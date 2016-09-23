import { combineReducers } from 'redux'
import actions from '../actions'

const success = (message) => ({ message: message, type: 'success', displayed: false })

export const flashMessage = (state = {}, action) => {
  if (action.type == actions.account.created.type) {
    return success('Created account')
  } else if (action.type == actions.asset.created.type) {
    return success('Created asset')
  } else if (action.type == actions.transaction.created.type) {
    return success('Created transaction')
  } else if (action.type == actions.mockhsm.created.type) {
    return success('Created key')
  } else if (action.type == actions.app.displayedFlash.type) {
    return Object.assign({}, state, { displayed: true })
  } else if (action.type == '@@router/LOCATION_CHANGE' && state.displayed == true) {
    if (action.payload.state && action.payload.state.preserveFlash) {
      return state
    }
    return {}
  } else if (action.type == actions.app.dismissFlash.type) {
    return {}
  }

  return state
}

export const dropdownState = (state = '', action) => {
  if (action.type == actions.app.toggleDropdown.type) {
    return state === '' ? 'open' : ''
  } else if (action.type == actions.app.closeDropdown.type) {
    return ''
  }

  return state
}

export default combineReducers({
  dropdownState,
  flashMessage
})
