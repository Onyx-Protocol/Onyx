import { combineReducers } from 'redux'
import moment from 'moment-timezone'

export const testNetInfo = (state = { loading: true }, action) => {
  if (action.type == 'TEST_NET_CONFIG') {
    state = {...action.data}
  }

  return state
}

export const testNetResetTime = (state = '', action) => {
  const startOfNextWeek = moment().tz('America/Los_Angeles').isoWeekday(7).startOf('day')

  if (action.type == 'TEST_NET_CONFIG') {
    if (action.data.next_reset) {
      return moment(action.data.next_reset)
    } else {
      return startOfNextWeek
    }
  }

  return state
}

export const testNetRpcVersion = (state = 0, action) => {
  if (action.type == 'TEST_NET_CONFIG') {
    return action.data.network_rpc_version
  }

  return state
}

export default combineReducers({
  testNetInfo,
  testNetResetTime,
  testNetRpcVersion
})

export const getUpcomingReset = (state) =>
  moment(state.configuration.testNetResetTime)
    .diff(moment(), 'days') < 1

export const getNetworkMismatch = (state) =>
  state.core.networkRpcVersion != 0 &&
  state.configuration.testNetRpcVersion != 0 &&
  state.core.networkRpcVersion != state.configuration.testNetRpcVersion
