import { client } from '../core'
import { getSourceMap } from './selectors'

export const loadTemplate = (selected: string) => {
  return (dispatch, getState) => {
    const state = getState()
    const source = getSourceMap(state)[selected]
    dispatch(setSource(source))
  }
}

export const SET_SOURCE = 'templates/SET_SOURCE'

export const setSource = (source: string) => {
  return (dispatch, getState) => {
    const type = SET_SOURCE
    dispatch({
      type,
      source
    })
    dispatch(fetchCompiled(source))
  }
}

export const FETCH_COMPILED = 'templates/FETCH_COMPILED'

export const fetchCompiled = (source: string) => {
  return (dispatch, getState) => {
    return client.ivy.compile({ contract: source }).then((compiled) => {
      let type = FETCH_COMPILED
      dispatch({ type, compiled })
    }).catch((e) => {throw e})
  }
}

export const SAVE_TEMPLATE = 'templates/SAVE_TEMPLATE'

export const saveTemplate = () => ({ type: SAVE_TEMPLATE })
