// ivy imports
import { client } from '../core'
import { generateInputMap } from '../contracts/selectors'

// internal imports
import { INITIAL_ID_LIST } from './constants'
import { getSourceMap, hasSourceChanged } from './selectors'
import { CompilerResult, CompiledTemplate } from './types'
import { makeEmptyTemplate, getDefaultContract, formatCompilerResult } from './util'

export const loadTemplate = (selected: string) => {
  return (dispatch, getState) => {
    if (!selected) {
      selected = INITIAL_ID_LIST[1]
    }
    const state = getState()
    const source = getSourceMap(state)[selected]
    dispatch(setSource(source))
  }
}

export const SHOW_LOCK_INPUT_ERRORS = 'templates/SHOW_LOCK_INPUT_ERRORS'

export const showLockInputErrors = (result: boolean) => {
  return {
    type: SHOW_LOCK_INPUT_ERRORS,
    result
  }
}

export const UPDATE_LOCK_ERROR = 'templates/UPDATE_LOCK_ERROR'

export const updateLockError = (error?) => {
  return {
    type: UPDATE_LOCK_ERROR,
    error
  }
}

export const SET_SOURCE = 'templates/SET_SOURCE'

export const setSource = (source: string) => {
  return (dispatch, getState) => {
    const type = SET_SOURCE
    const sourceChanged = hasSourceChanged(source)(getState())
    dispatch({ type, source, sourceChanged })
    dispatch(fetchCompiled(source))
    dispatch(updateLockError())
  }
}

export const FETCH_COMPILED = 'templates/FETCH_COMPILED'

export const fetchCompiled = (source: string) => {
  return (dispatch, getState) => {
    const type = FETCH_COMPILED
    const sourceMap = getSourceMap(getState())
    return client.ivy.compile({ source }).then((result: CompilerResult) => {
      if (result.error) {
        const compiled: CompiledTemplate =  makeEmptyTemplate(source, result.error)
        const inputMap = {}
        return dispatch({ type, compiled, inputMap })
      }

      const formatted: CompilerResult = formatCompilerResult(result)
      const compiled: CompiledTemplate = getDefaultContract(source, formatted)
      const inputMap = generateInputMap(compiled)
      return dispatch({ type, compiled, inputMap })
    }).catch((e) => {throw e})
  }
}

export const SAVE_TEMPLATE = 'templates/SAVE_TEMPLATE'

export const saveTemplate = () => ({ type: SAVE_TEMPLATE })
