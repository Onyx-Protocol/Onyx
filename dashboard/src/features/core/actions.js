import chain from 'chain'
import { context } from 'utility/environment'
import actionCreator from 'actions/actionCreator'

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
          dispatch({type: 'ERROR', payload: err})
        }
      })
  }
}

const retry = (dispatch, promise, count = 10) => {
  return dispatch(promise).catch((err) => {
    var currentTime = new Date().getTime()
    while (currentTime + 200 >= new Date().getTime()) { /* wait for retry */ }

    if (count >= 1) {
      retry(dispatch, promise, count - 1)
    } else {
      throw(err)
    }
  })
}

let actions = {
  setClientToken,
  updateInfo,
  fetchCoreInfo,
  userLoggedIn,
  clearSession,
  logIn: (token) => (dispatch) => {
    dispatch(setClientToken(token))
    return dispatch(fetchCoreInfo({throw: true})).then(
      () => dispatch(userLoggedIn())
    )
  },
  submitConfiguration: (data) => {
    return (dispatch) => {
      // Convert string value to boolean for API
      data.is_generator = data.is_generator === 'true' ? true : false
      data.is_signer = data.is_generator
      data.quorum = data.is_generator ? 1 : 0

      return chain.Core.configure(context(), data)
        .then(() => retry(dispatch, fetchCoreInfo({throw: true})))
        .catch((err) => dispatch({type: 'ERROR', payload: err}))
    }
  }
}

export default actions
