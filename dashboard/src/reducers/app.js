import { combineReducers } from 'redux'
import actions from '../actions'

export const dropdownState = (state = "", action) => {
  if (action.type == actions.app.toggleDropdown.type) {
    return state === "" ? "open" : ""
  } else if (action.type == actions.app.closeDropdown.type) {
    return ""
  }

  return state
}

export default combineReducers({
  dropdownState
})
