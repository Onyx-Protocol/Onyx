export const exportState = (store) => () => {
  localStorage.setItem('reduxState', JSON.stringify(store.getState().core))
}
export const importState = () => {
  const state = localStorage.getItem('reduxState') ?
    JSON.parse(localStorage.getItem('reduxState')) :
    {}

  return {
    core: {
      clientToken: state.clientToken,
      validToken: state.validToken,
    }
  }
}
