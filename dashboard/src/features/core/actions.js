import { chainClient } from 'utility/environment'

const updateInfo = (param) => ({type: 'UPDATE_CORE_INFO', param})

export const fetchCoreInfo = (options = {}) => {
  return (dispatch) => {
    return chainClient().config.info()
      .then(
        (info) => dispatch(updateInfo(info)),
        (err) => {
          if (options.throw || !err.status) {
            throw err
          } else {
            if (err.status == 401) dispatch({type: 'AUTHENTICATION_INVALID'})
            else                   dispatch({type: 'CORE_DISCONNECT'})
          }
        }
      )
  }
}

let actions = {
  updateInfo,
  fetchCoreInfo,
}

export default actions
