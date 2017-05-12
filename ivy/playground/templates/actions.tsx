import { client } from '../util'
import { getItemMap, getIdList, getCompiled, getContractParameters } from './selectors'
import { Template, ItemMap, CompilerResult } from './types'
import { ContractParameter, TemplateClause, ClauseParameter } from 'ivy-compiler'
import { mapServerTemplate } from './util'
import { generateInputMap } from '../contracts/reducer'

export const SET_INITIAL_TEMPLATES = 'templates/SET_TEMPLATES'
export const SET_SOURCE = 'templates/SET_SOURCE'
export const SET_COMPILED = 'templates/SET_COMPILED'
export const COMPILER_ERROR = 'templates/COMPILER_ERROR'

export const load = (selected: string) => {
  return (dispatch, getState) => {
    const state = getState()
    const source = getItemMap(state)[selected]
    dispatch(fetchCompiled(source))
  }
}

export const fetchCompiled = (source: string) => {
  return (dispatch, getState) => {
    return client.ivy.compile({ contract: source }).then((res) => {
      let tpl
      if (res.error === "") {
        tpl = mapServerTemplate(res)
      }
      tpl = {...tpl, error: res.error}
      if (tpl.instructions) {
        tpl.opcodes = tpl.instructions.join(' ')
      }

      let type = SET_COMPILED
      // Error results return a blank source so we
      // pass the compiled source to the reducer.
      const compiled = { ...tpl, source}
      dispatch({ type, compiled })

      type = SET_SOURCE
      const contractParameters = compiled.contractParameters
      const inputMap = contractParameters ? generateInputMap(contractParameters, res.value) : undefined
      dispatch({ type, source, contractParameters, inputMap })
    }).catch((e) => {throw e})
  }
}

export const setSource = (source: string) => {
  return (dispatch, getState) => {
    const type = SET_SOURCE
    dispatch(fetchCompiled(source))
  }
}

export const SAVE_TEMPLATE = 'SAVE_TEMPLATE'

export function saveTemplate() {
  return {
    type: SAVE_TEMPLATE
  }
}
