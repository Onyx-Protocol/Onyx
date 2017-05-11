import { compileContractParameters, ContractParameter } from 'ivy-compiler';
import { TemplateState, CompilerResult } from './types'
import { SET_INITIAL_TEMPLATES, SET_SOURCE, 
         SAVE_TEMPLATE, SET_COMPILED } from './actions'
import { UPDATE_INPUT } from '../contracts/actions'
import { INITIAL_STATE } from './constants'

export default function reducer(state: TemplateState = INITIAL_STATE, action): TemplateState {
  switch (action.type) {
    case UPDATE_INPUT:
      let name = action.name
      let newValue = action.newValue
      if (state.inputMap === undefined) return state
      return {
        ...state,
        inputMap: {
          ...state.inputMap,
          [name]: {
            ...state.inputMap[name],
            value: newValue
          }
        }
      }
    case SET_SOURCE: {
      const source = action.source
      const contractParameters = action.contractParameters
      const inputMap = action.inputMap
      return {
        ...state,
        source,
        inputMap,
        contractParameters
      }
    }
    case SAVE_TEMPLATE: {
      let compiled = state.compiled
      if ((compiled === undefined) || 
          (compiled.error !== "") || 
          (state.itemMap[compiled.name] !== undefined)) return state // this shouldn't happen
      return {
        ...state,
        idList: [...state.idList, compiled.name],
        itemMap: {
          ...state.itemMap,
          [compiled.name]: compiled.source
        }
      }
    }
    case SET_COMPILED: {
      const compiled = action.compiled
      const error = compiled.error
      if (error !== undefined && compiled.source == state.source) {
        // the JS compiler may be less strict than the Go compiler
        // so this should make sure the contractParameters and inputMap are not rendered
        // I think this won't be a race condition
        return {
          ...state,
          inputMap: undefined,
          contractParameters: undefined,
          compiled
        }
      }
      if (state.itemMap === undefined) {
        // the JS compiler should never be STRICTER than the Go compiler
        throw "compilation should never succeed if typechecking fails"
      }
      return {
        ...state,
        compiled
      }
    }
    default:
      return state
  }
}
