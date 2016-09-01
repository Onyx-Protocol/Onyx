import { push } from 'react-router-redux'

let actions = {
  showRoot: push('/transactions'),
  showConfiguration: () => {
    return (dispatch, getState) => {
      let pathname = getState().routing.locationBeforeTransitions.pathname
      if (pathname !== 'configuration') {
        dispatch(push('/configuration'))        
      }
    }
  }
}

export default actions
