export const exportState = (store) => () => {
  const state = store.getState()
  const exportable = {
    core: state.core,
    transaction: {
      generated: (state.transaction || {}).generated,
    },
  }

  localStorage.setItem('reduxState', JSON.stringify(exportable))
}

export const importState = () => {
  const state = localStorage.getItem('reduxState')
  if (!state) return {}

  try {
    return JSON.parse(state)
  } catch (_) {
    return {}
  }
}
