const actions = {
  showModal: (body, accept, cancel, options = {}) => ({type: 'SHOW_MODAL', payload: { body, accept, cancel, options }}),
  hideModal: { type: 'HIDE_MODAL' },
  toggleDropdown: { type: 'TOGGLE_DROPDOWN' },
  closeDropdown: (dispatch, getState) => {
    if (getState().app.dropdownState === 'open') {
      dispatch({ type: 'CLOSE_DROPDOWN' })
    }
  },
}

export default actions
