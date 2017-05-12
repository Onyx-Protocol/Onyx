import { client } from '../core'
import { getItemMap, getIdList, getCompiled, getContractParameters } from './selectors'
import { Template, ItemMap, CompilerResult } from './types'
import { ContractParameter, TemplateClause, ClauseParameter } from 'ivy-compiler'
import { mapServerTemplate } from './util'

export const SET_INITIAL_TEMPLATES = 'templates/SET_TEMPLATES'
export const SET_SOURCE = 'templates/SET_SOURCE'
export const SET_COMPILED = 'templates/SET_COMPILED'
export const COMPILER_ERROR = 'templates/COMPILER_ERROR'

export const load = (selected: string) => {
  return (dispatch, getState) => {
    const state = getState()
    const source = getItemMap(state)[selected]
    dispatch(setSource(source))
  }
}

export const fetchCompiled = (source: string) => {
  return (dispatch, getState) => {
    return client.ivy.compile({ contract: source }).then((compiled) => {
      let type = SET_COMPILED
      dispatch({ type, compiled })
    }).catch((e) => {throw e})
  }
}

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

export const SAVE_TEMPLATE = 'SAVE_TEMPLATE'

export function saveTemplate() {
  return {
    type: SAVE_TEMPLATE
  }
}
