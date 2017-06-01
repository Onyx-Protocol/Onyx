import { unauthedClient } from 'utility/environment'
import { fetchCoreInfo } from 'features/core/actions'

const setClientToken = (token) => ({type: 'SET_CLIENT_TOKEN', token})

export const clearSession = ({ type: 'USER_LOG_OUT' })

export const logIn = (token) => (dispatch) => {
  dispatch(setClientToken(token))
  return dispatch(fetchCoreInfo({throw: true}))
    .then(() => dispatch({type: 'USER_LOG_IN'})
  )
}

export const getAuthenticationStatus = () => {
  return (dispatch) => {
    return unauthedClient().config.info()
      .then(() => {
        dispatch({type: 'CONNECTED_TO_CORE'})
      }, (err) => {
        if (err.status < 500) {
          dispatch({type: 'CONNECTED_TO_CORE'})
        } else {
          dispatch({type: 'CORE_DISCONNECT'})
        }

        if (err.status == 401) {
          dispatch({type: 'AUTHENTICATION_REQUIRED'})
        }
      })
      .then(() => dispatch(fetchCoreInfo()))
      .then(() => dispatch({type: 'AUTHN_LOADED'}))
  }
}
