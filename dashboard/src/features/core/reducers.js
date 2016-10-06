import { combineReducers } from 'redux'
import { testNetUrl } from 'utility/environment'
import moment from 'moment'

const LONG_TIME_FORMAT = 'YYYY-MM-DD, h:mm:ss a'

const coreConfigReducer = (key, state, defaultState, action) => {
  if (action.type == 'UPDATE_CORE_INFO') {
    return action.param[key] || defaultState
  }

  return state || defaultState
}

export const configured = (state, action) =>
  coreConfigReducer('is_configured', state, false, action)
export const configuredAt = (state, action) => {
  let value = coreConfigReducer('configured_at', state, '', action)
  if (action.type == 'UPDATE_CORE_INFO' && value != '') {
    value = moment(value).format(LONG_TIME_FORMAT)
  }
  return value
}
export const buildCommit = (state, action) => {
  let value = coreConfigReducer('build_commit', state, '', action)
  if (value === '?') {
    value = 'Local development'
  } else if (value != '') {
    value = value.substring(0,18)
  }
  return value
}
export const buildDate = (state, action) => {
  let value = coreConfigReducer('build_date', state, '', action)
  if (value !== '') {
    value = moment(value, 'X').format(LONG_TIME_FORMAT)
  }

  return value
}
export const production = (state, action) =>
  coreConfigReducer('is_production', state, false, action)
export const blockHeight = (state, action) =>
  coreConfigReducer('block_height', state, 0, action)
export const generatorBlockHeight = (state, action) =>
  coreConfigReducer('generator_block_height', state, 0, action)
export const generator = (state, action) =>
  coreConfigReducer('is_generator', state, false, action)
export const generatorUrl = (state, action) =>
  coreConfigReducer('generator_url', state, false, action)
export const generatorAccessToken = (state, action) =>
  coreConfigReducer('generator_access_token', state, false, action)
export const blockchainID = (state, action) =>
  coreConfigReducer('blockchain_id', state, 0, action)
export const networkRpcVersion = (state, action) =>
  coreConfigReducer('network_rpc_version', state, 0, action)
export const requireNetworkToken = (state, action) =>
  coreConfigReducer('require_network_access_tokens', state, false, action)

export const replicationLag = (state = null, action) => {
  if (action.type == 'UPDATE_CORE_INFO') {
    return action.param.generator_block_height - action.param.block_height + ''
  }

  return state
}

export const onTestNet = (state = false, action) => {
  if (action.type == 'UPDATE_CORE_INFO') {
    return action.param.generator_url == testNetUrl
  }

  return state
}

export const requireClientToken = (state, action) => {
  let value = coreConfigReducer('require_client_access_tokens', state, false, action)
  if (action.type == 'ERROR' && action.payload.status == 401) return true

  return value
}

export const clientToken = (state = '', action) => {
  if      (action.type == 'SET_CLIENT_TOKEN') return action.token
  else if (action.type == 'USER_LOG_OUT')     return ''
  else if (action.type == 'UPDATE_CORE_INFO' &&
           action.param.require_client_access_tokens == false) return ''
  else if (action.type == 'ERROR' &&
           action.payload.status == 401)      return ''

  return state
}

export const validToken = (state = false, action) => {
  if      (action.type == 'SET_CLIENT_TOKEN') return false
  else if (action.type == 'USER_LOG_IN')      return true
  else if (action.type == 'USER_LOG_OUT')     return false
  else if (action.type == 'ERROR' &&
           action.payload.status == 401)      return false

  return state
}

export default combineReducers({
  configured,
  configuredAt,
  production,
  buildCommit,
  buildDate,
  blockHeight,
  generatorBlockHeight,
  replicationLag,
  generator,
  generatorUrl,
  generatorAccessToken,
  blockchainID,
  requireClientToken,
  clientToken,
  validToken,
  requireNetworkToken,
  onTestNet,
  networkRpcVersion,
})
