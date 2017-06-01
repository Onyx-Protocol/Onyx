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

      // TODO: If the dashboard has a way of probing the core for a token
      // requirement, we won't need to store these anymore.
      // requireClientToken: (state.core || {}).requireClientToken,
      // validToken: (state.core || {}).validToken,
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
