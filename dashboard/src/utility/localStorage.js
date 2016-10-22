export const exportState = (store) => () => {
  const state = store.getState()
  const exportable = {
    core: state.core,
    transaction: {
      generated: (state.transaction || {}).generated,
    },
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
