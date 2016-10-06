import chain from 'chain'
import { context } from 'utility/environment'
import { actions as coreActions } from 'features/core'
import actionCreator from 'actions/actionCreator'
import { testNetInfoUrl } from 'utility/environment'

const receivedTestNetConfig = actionCreator('TEST_NET_CONFIG', data => ({ data }))

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
  fetchTestNetInfo: () => {
    return (dispatch) => {
      fetch(testNetInfoUrl)
        .then(resp => resp.json())
        .then(json => dispatch(receivedTestNetConfig(json)))
    }
  },
  submitConfiguration: (data) => {
    return (dispatch) => {

      if (data.type == 'new') {
        data = {
          is_generator: true,
          is_signer: true,
          quorum: 1,
        }
      }

      delete data.type

      return chain.Core.configure(context(), data)
        .then(() => retry(dispatch, coreActions.fetchCoreInfo({throw: true})))
    }
  }
}

export default actions
