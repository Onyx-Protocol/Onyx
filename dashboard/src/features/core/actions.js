import chain from '_chain'
import { context } from 'utility/environment'

const updateInfo = (param) => ({type: 'UPDATE_CORE_INFO', param})
const setClientToken = (param) => ({type: 'SET_CLIENT_TOKEN', param})
const clearSession = ({ type: 'USER_LOG_OUT' })

const fetchCoreInfo = (options = {}) => {
  return (dispatch) => {
    return chain.Core.info(context())
      .then((info) => dispatch(updateInfo(info)))
      .catch((err) => {
        if (options.throw || !chain.errors.isChainError(err)) {
          throw err
        } else {
          if (err.status == 401) {
            dispatch({type: 'ERROR', payload: err})
          } else {
            dispatch({type: 'CORE_DISCONNECT'})
          }
        }
      })
  }
}

let actions = {
  setClientToken,
  updateInfo,
  fetchCoreInfo,
  clearSession,
  logIn: (token) => (dispatch) => {
    dispatch(setClientToken(token))
    return dispatch(fetchCoreInfo({throw: true}))
      .then(() => dispatch({type: 'USER_LOG_IN'})
    )
  }
}

export default actions
