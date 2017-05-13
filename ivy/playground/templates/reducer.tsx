// external imports
import { compileContractParameters, ContractParameter } from 'ivy-compiler';

// ivy imports
import { TemplateState, CompiledTemplate } from './types'
import { UPDATE_INPUT } from '../contracts/actions'
import { generateInputMap } from '../contracts/selectors'
import { InputMap } from '../inputs/types'

// internal imports
import { SET_SOURCE, SAVE_TEMPLATE, FETCH_COMPILED } from './actions'
import { INITIAL_SOURCE_MAP, INITIAL_ID_LIST } from './constants'

const INITIAL_STATE: TemplateState = {
  sourceMap: INITIAL_SOURCE_MAP,
  idList: INITIAL_ID_LIST,
  source: '',
  inputMap: undefined,
  compiled: undefined
}

export default function reducer(state: TemplateState = INITIAL_STATE, action): TemplateState {
  switch (action.type) {
    case UPDATE_INPUT: {
      const name = action.name
      const newValue = action.newValue
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
    }
    case SET_SOURCE: {
      const source = action.source
      return {
        ...state,
        source
      }
    }
    case SAVE_TEMPLATE: {
      const compiled = state.compiled
      if ((compiled === undefined) ||
          (compiled.error !== "") ||
          (state.sourceMap[compiled.name] !== undefined)) return state // this shouldn't happen
      return {
        ...state,
        idList: [...state.idList, compiled.name],
        sourceMap: {
          ...state.sourceMap,
          [compiled.name]: compiled.source
        }
      }
    }
    case FETCH_COMPILED: {
      const compiled: CompiledTemplate = action.compiled
      const inputMap: InputMap = action.inputMap
      return {
        ...state,
        compiled,
        inputMap
      }
    }
    default:
      return state
  }
}
