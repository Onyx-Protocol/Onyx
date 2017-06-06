export const clear = () => {
  try {
    localStorage.clear()
  } catch (err) {
    // Local storage is not available.
  }
}

export const exportState = (store) => () => {
  const state = store.getState()
  const exportable = {
    authn: {
      clientToken: (state.authn || {}).clientToken,
    },
    transaction: {
      generated: (state.transaction || {}).generated,
    },
    tutorial: state.tutorial
  }

  try {
    localStorage.setItem('reduxState', JSON.stringify(exportable))
  } catch (err) { /* localstorage not available */ }
}

export const importState = () => {
  let state
  try {
    state = localStorage.getItem('reduxState')
  } catch (err) { /* localstorage not available */ }

  if (!state) return {}

  try {
    return JSON.parse(state)
  } catch (_) {
    return {}
  }
}
