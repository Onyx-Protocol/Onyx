import { combineReducers } from 'redux'
import moment from 'moment-timezone'

export const nextReset = (state = '', action) => {
  if (action.type == 'TESTNET_CONFIG') {
    if (action.data.next_reset) {
      return moment(action.data.next_reset)
    } else {
      // Default reset time is the upcoming Sunday 00:00:00 Pacific.
      return moment().tz('America/Los_Angeles').day(7).startOf('day')
    }
  }
  return state
}

export const blockchainId = (state = 0, action) => {
  if (action.type == 'TESTNET_CONFIG') {
    return action.data.blockchain_id
  }
  return state
}

export const crosscoreRpcVersion = (state = 0, action) => {
  if (action.type == 'TESTNET_CONFIG') {
    return action.data.crosscore_rpc_version || action.data.network_rpc_version
  }
  return state
}

export const testnetInfo = (state = { loading: true }, action) => {
  if (action.type == 'TESTNET_CONFIG') {
    state = {...action.data}
  }
  return state
}

export default combineReducers({
  blockchainId,
  nextReset,
  crosscoreRpcVersion,
  testnetInfo,
})
