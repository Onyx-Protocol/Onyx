import { combineReducers } from 'redux'
import actions from '../actions'

const coreConfigReducer = (key, state, defaultState, action) => {
  if (action.type == actions.core.updateInfo.type) {
    return action.param[key] || defaultState
  }

  return state || defaultState
}

export const configured = (state, action) =>
  coreConfigReducer('is_configured', state, false, action)
export const configuredAt = (state, action) =>
  coreConfigReducer('configured_at', state, "", action)
export const buildCommit = (state, action) =>
  coreConfigReducer('build_commit', state, "", action)
export const buildDate = (state, action) =>
  coreConfigReducer('build_date', state, "", action)
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
export const initialBlockHash = (state, action) =>
  coreConfigReducer('initial_block_hash', state, 0, action)


export const replicationLag = (state = null, action) => {
  if (action.type == actions.core.updateInfo.type) {
    return action.param.generator_block_height - action.param.block_height + "" 
  }

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
  initialBlockHash
})
