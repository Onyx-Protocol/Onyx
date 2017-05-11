import { client } from '../util'
import { getItemMap, getIdList, getCompiled, getContractParameters } from './selectors'
import { Template, ItemMap, CompilerResult } from './types'
import { ContractParameter, TemplateClause, ClauseParameter } from 'ivy-compiler'

export const SET_INITIAL_TEMPLATES = 'templates/SET_TEMPLATES'
export const SET_SOURCE = 'templates/SET_SOURCE'
export const SET_COMPILED = 'templates/SET_COMPILED'
export const COMPILER_ERROR = 'templates/COMPILER_ERROR'

export const load = (selected: string) => {
  return (dispatch, getState) => {
    let state = getState()
    let source = getItemMap(state)[selected]
    dispatch({
      type: SET_SOURCE,
      source: source
    })
    dispatch(fetchCompiled(source))
  }
}


export const fetchCompiled = (source: string) => {
  return (dispatch, getState) => {
    return client.ivy.compile({ contract: source }).then(
      (compiled: CompilerResult) => dispatch({type: SET_COMPILED, result: compiled})
    ).catch((e) => {throw e})
  }
}

export const setSource = (source: string) => {
  return (dispatch, getState) => {
    const type = SET_SOURCE
    dispatch(fetchCompiled(source))
    return dispatch({ type, source })
  }
}

export const SAVE_TEMPLATE = 'SAVE_TEMPLATE'

export function saveTemplate() {
  return {
    type: SAVE_TEMPLATE
  }
}
