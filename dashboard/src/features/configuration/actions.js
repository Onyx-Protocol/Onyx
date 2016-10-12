// FIXME: Microsoft Edge has issues returning errors for responses
// with a 401 status. We should add browser detection to only
// use the ponyfill for unsupported browsers.
const { fetch } = require('fetch-ponyfill')()

import chain from 'chain'
import { context } from 'utility/environment'
import { actions as coreActions } from 'features/core'
import { actionCreator } from 'features/shared/actions'
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

const fetchTestNetInfo = () => {
  return (dispatch) =>
    fetch(testNetInfoUrl)
      .then(resp => resp.json())
      .then(json => {
        dispatch(receivedTestNetConfig(json))
        return json
      })
}

let actions = {
  fetchTestNetInfo,
  submitConfiguration: (data) => {
    const configureWithRetry = (dispatch, config) => {
      return chain.Core.configure(context(), config)
        .then(() => retry(dispatch, coreActions.fetchCoreInfo({throw: true})))
    }

    return (dispatch) => {
      if (data.type == 'testnet') {
        return dispatch(fetchTestNetInfo()).then(testNet =>
          configureWithRetry(dispatch, testNet))
      } else {
        if (data.type == 'new') {
          data = {
            is_generator: true,
            is_signer: true,
            quorum: 1,
          }
        }

        delete data.type
        return configureWithRetry(dispatch, data)
      }
    }
  }
}

export default actions
