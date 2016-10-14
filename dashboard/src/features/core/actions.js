import chain from 'chain'
import { context } from 'utility/environment'
import { actionCreator } from 'features/shared/actions'

const updateInfo = actionCreator('UPDATE_CORE_INFO', param => ({ param }))
const setClientToken = actionCreator('SET_CLIENT_TOKEN', token => ({ token }))
const userLoggedIn = actionCreator('USER_LOG_IN')
const clearSession = actionCreator('USER_LOG_OUT')

const fetchCoreInfo = (options = {}) => {
  return (dispatch) => {
    return chain.Core.info(context())
      .then((info) => dispatch(updateInfo(info)))
      .catch((err) => {
        if (options.throw) {
          throw err
        } else {
          dispatch({type: 'CORE_DISCONNECT'})
        }
      })
  }
}

let actions = {
  setClientToken,
  updateInfo,
  fetchCoreInfo,
  userLoggedIn,
  clearSession,
  logIn: (token) => (dispatch) => {
    dispatch(setClientToken(token))
    return dispatch(fetchCoreInfo({throw: true}))
      .then(() => dispatch(userLoggedIn())
    )
  }
}

export default actions
