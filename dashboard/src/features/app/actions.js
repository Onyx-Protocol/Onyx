const actions = {
  showModal: (title, body, options = {}) => ({type: 'SHOW_MODAL', payload: { title, body, options }}),
  toggleDropdown: { type: 'TOGGLE_DROPDOWN' },
  closeDropdown: (dispatch, getState) => {
    if (getState().app.dropdownState === 'open') {
      dispatch({ type: 'CLOSE_DROPDOWN' })
    }
  },
}

export default actions
