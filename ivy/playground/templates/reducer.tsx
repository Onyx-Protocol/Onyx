import { compileContractParameters, ContractParameter } from 'ivy-compiler';
import { TemplateState } from './types'
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
      return { 
        ...state,
        compiled: action.result
      }
    }
    default:
      return state
  }
}
