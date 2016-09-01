import actionCreator from './actionCreator'

const toggleDropdown = actionCreator(`TOGGLE_DROPDOWN`)
const _closeDropwdown = actionCreator(`CLOSE_DROPDOWN`)

const closeDropdown = () => (dispatch, getState) => {
  if (getState().app.dropdownState === "open") {
    dispatch(_closeDropwdown())
  }
}
closeDropdown.type = _closeDropwdown.type

export default {
  toggleDropdown,
  closeDropdown
}
