import chain from '_chain'
import { context } from 'utility/environment'
import { chainClient } from 'utility/environment'

const updateInfo = (param) => ({type: 'UPDATE_CORE_INFO', param})
const setClientToken = (token) => ({type: 'SET_CLIENT_TOKEN', token})
const clearSession = ({ type: 'USER_LOG_OUT' })

const fetchCoreInfo = (options = {}) => {
  return (dispatch) => {
    return chainClient().config.info()
      .then((info) => dispatch(updateInfo(info)))
      .catch((err) => {
        if (options.throw || !err.status) {
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
