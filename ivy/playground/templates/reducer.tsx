import { compileContractParameters, ContractParameter } from 'ivy-compiler';
import { TemplateState, CompilerResult } from './types'
import { SET_INITIAL_TEMPLATES, SET_SOURCE, 
         SAVE_TEMPLATE, SET_COMPILED } from './actions'
import { UPDATE_INPUT } from '../contracts/actions'
import { INITIAL_STATE } from './constants'
import { generateInputMap } from '../contracts/reducer'

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
      let source = action.source
      let contractParameters: undefined | ContractParameter[] = undefined
      try {
        contractParameters = compileContractParameters(source)
      } catch(e) {
        console.log("typecheck error", e) // the Go compiler is responsible for returning the error that actually gets presented
      }
      const inputMap = contractParameters ? generateInputMap(contractParameters) : undefined
      return {
        ...state,
        source: source,
        inputMap: inputMap,
        contractParameters: contractParameters
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
      let compiled = action.result
      if (compiled.error) { 
        // the JS compiler may be less strict than the Go compiler
        // so this should make sure the contractParameters and inputMap are not rendered
        // the Create section was generated synchronously and this is asynchronous
        // so this shouldn't be a race condition
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
