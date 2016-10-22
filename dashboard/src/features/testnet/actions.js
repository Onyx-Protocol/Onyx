// FIXME: Microsoft Edge has issues returning errors for responses
// with a 401 status. We should add browser detection to only
// use the ponyfill for unsupported browsers.
const { fetch } = require('fetch-ponyfill')()
import { testnetInfoUrl } from 'utility/environment'

export const fetchTestnetInfo = () => {
  return (dispatch) =>
    fetch(testnetInfoUrl)
      .then(resp => resp.json())
      .then(json => {
        dispatch({type: 'TESTNET_CONFIG', data: json})
        return json
      })
}

const actions = {
  fetchTestnetInfo,
}

export default actions
