import chain from '../chain'
import { context } from '../utility/environment'
import actionCreator from './actionCreator'
import routing from './routing'

const updateInfo = actionCreator(`UPDATE_CORE_INFO`, param => { return { param }})

let actions = {
  updateInfo,
  fetchCoreInfo: () => {
    return (dispatch) => {
      return chain.Core.info(context)
        .then((info) => dispatch(updateInfo(info)))
    }
  },
  submitConfiguration: (data) => {
    return (dispatch) => {
      // Convert string value to boolean for API
      data.is_generator = data.is_generator === 'true' ? true : false
      data.is_signer = data.is_generator

      let configError

      return chain.Core.configure(data, context)
        .then(() => chain.Core.info(context))
        .catch((err) => {
          configError = err

          console.log("Sleep 2s for core to come back")
          var currentTime = new Date().getTime()
          while (currentTime + 2000 >= new Date().getTime()) {}

          return chain.Core.info(context)
        })
        .then((info) => {
          dispatch(updateInfo(info))
          if (info.is_configured) {
            dispatch(routing.showRoot)
          } else {
            throw(configError)
          }
        })
    }
  }
}

export default actions
