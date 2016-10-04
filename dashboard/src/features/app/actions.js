import actionCreator from 'actions/actionCreator'

const actions = {
  dismissFlash: actionCreator('DISMISS_FLASH', param => ({ param })),
  displayedFlash: actionCreator('DISPLAYED_FLASH', param => ({ param })),
  showModal: actionCreator('SHOW_MODAL', (body, accept, cancel, options = {}) =>
    ({ payload: { body, accept, cancel, options }})),
  hideModal: actionCreator('HIDE_MODAL'),
}

export default actions
