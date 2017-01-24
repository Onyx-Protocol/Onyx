import { push } from 'react-router-redux'

const actions = {
  dismissFlash: (param) => ({type: 'DISMISS_FLASH', param}),
  displayedFlash: (param) => ({type: 'DISPLAYED_FLASH', param}),
  showModal: (body, accept, cancel, options = {}) => ({type: 'SHOW_MODAL', payload: { body, accept, cancel, options }}),
  hideModal: { type: 'HIDE_MODAL' },
  showRoot: push('/transactions'),
  toggleDropdown: { type: 'TOGGLE_DROPDOWN' },
  closeDropdown: (dispatch, getState) => {
    if (getState().app.dropdownState === 'open') {
      dispatch({ type: 'CLOSE_DROPDOWN' })
    }
  },
  showConfiguration: () => {
    return (dispatch, getState) => {
      // Need a default here, since locationBeforeTransitions gets cleared
      // during logout.
      let pathname = (getState().routing.locationBeforeTransitions || {}).pathname

      if (pathname !== 'configuration') {
        dispatch(push('/configuration'))
      }
    }
  }
}

export default actions
