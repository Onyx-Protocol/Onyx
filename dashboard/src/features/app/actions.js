import actionCreator from '../../actions/actionCreator'

const actions = {
  dismissFlash: actionCreator('DISMISS_FLASH', param => ({ param })),
  displayedFlash: actionCreator('DISPLAYED_FLASH', param => ({ param }))
}

export default actions
