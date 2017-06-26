import { unauthedClient } from 'utility/environment'
import { fetchCoreInfo } from 'features/core/actions'

export const logIn = (token) => (dispatch) => {
  dispatch({type: 'SET_CLIENT_TOKEN', token})
  return dispatch(fetchCoreInfo({throw: true}))
    .then(() => dispatch({type: 'USER_LOG_IN'})
  )
}

export const getAuthenticationStatus = () => {
  let connectedToCore = false
  const pingAuth = (dispatch) => {
    return unauthedClient().config.info()
      .then(() => {
        connectedToCore = true
        dispatch({type: 'CONNECTED_TO_CORE'})
      }, (err) => {
        if (err.status < 500) {
          connectedToCore = true
          dispatch({type: 'CONNECTED_TO_CORE'})
        } else {
          dispatch({type: 'CORE_DISCONNECT'})
          setTimeout(() => pingAuth(dispatch), 1000)
        }

        if (err.status == 401) {
          dispatch({type: 'AUTHENTICATION_REQUIRED'})
        }
      })
      .then(() => connectedToCore && dispatch(fetchCoreInfo()))
      .then(() => connectedToCore && dispatch({type: 'AUTHENTICATION_READY'}))
  }

  return pingAuth
}
