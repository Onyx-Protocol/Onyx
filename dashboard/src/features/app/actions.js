import { actionCreator } from 'features/shared/actions'
import { push } from 'react-router-redux'

const actions = {
  dismissFlash: actionCreator('DISMISS_FLASH', param => ({ param })),
  displayedFlash: actionCreator('DISPLAYED_FLASH', param => ({ param })),
  showModal: actionCreator('SHOW_MODAL', (body, accept, cancel, options = {}) =>
    ({ payload: { body, accept, cancel, options }})),
  hideModal: actionCreator('HIDE_MODAL'),
  showRoot: push('/transactions'),
  toggleDropdown: { type: 'TOGGLE_DROPDOWN' },
  closeDropdown: (dispatch, getState) => {
    if (getState().app.dropdownState === 'open') {
      dispatch({ type: 'CLOSE_DROPDOWN' })
    }
  },
  showConfiguration: () => {
    return (dispatch, getState) => {
      let pathname = getState().routing.locationBeforeTransitions.pathname
      if (pathname !== 'configuration') {
        dispatch(push('/configuration'))
      }
    }
  }
}

export default actions
