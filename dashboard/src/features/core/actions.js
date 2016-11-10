import chain from '_chain'
import { context } from 'utility/environment'
import { actionCreator } from 'features/shared/actions'

const updateInfo = actionCreator('UPDATE_CORE_INFO', param => ({ param }))
const setClientToken = actionCreator('SET_CLIENT_TOKEN', token => ({ token }))
const clearSession = actionCreator('USER_LOG_OUT')

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
