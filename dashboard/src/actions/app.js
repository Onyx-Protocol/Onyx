import actionCreator from './actionCreator'

const _closeDropwdown = actionCreator('CLOSE_DROPDOWN')

const closeDropdown = () => (dispatch, getState) => {
  if (getState().app.dropdownState === 'open') {
    dispatch(_closeDropwdown())
  }
}
closeDropdown.type = _closeDropwdown.type

export default {
  toggleDropdown: actionCreator('TOGGLE_DROPDOWN'),
  closeDropdown,
  dismissFlash: actionCreator('DISMISS_FLASH', param => ({ param })),
  displayedFlash: actionCreator('DISPLAYED_FLASH', param => ({ param }))
}
