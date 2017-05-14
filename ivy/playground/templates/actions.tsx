// ivy imports
import { client } from '../core'
import { generateInputMap } from '../contracts/selectors'

// internal imports
import { getSourceMap } from './selectors'
import { CompiledTemplate } from './types'

export const loadTemplate = (selected: string) => {
  return (dispatch, getState) => {
    const state = getState()
    const source = getSourceMap(state)[selected]
    dispatch(setSource(source))
  }
}

export const DISPLAY_ERROR = 'templates/DISPLAY_ERROR'

export const displayError = (error) => {
  return {
    type: DISPLAY_ERROR,
    error
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
    return client.ivy.compile({ contract: source }).then((result) => {
      const type = FETCH_COMPILED
      const format = (tpl: CompiledTemplate) => {
        if (tpl.error !== '') {
          tpl.clauseInfo = tpl.params = []
        }
        return tpl
      }
      const compiled = format(result)
      const inputMap = generateInputMap(compiled)
      dispatch({ type, compiled, inputMap })
    }).catch((e) => {throw e})
  }
}

export const SAVE_TEMPLATE = 'templates/SAVE_TEMPLATE'

export const saveTemplate = () => ({ type: SAVE_TEMPLATE })
