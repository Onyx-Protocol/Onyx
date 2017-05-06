import { State } from './types'
import { COMPILE_TEMPLATES, LOAD_TEMPLATE, SET_SOURCE, SAVE_TEMPLATE } from './actions'
import { INITIAL_STATE } from './constants'
import { isError } from './util'

export default function reducer(state: State = INITIAL_STATE, action): State {
  switch (action.type) {
    case COMPILE_TEMPLATES:
      return {
        ...state,
        itemMap: action.itemMap,
        idList: action.idList,
        selected: action.selected,
        source: action.source
      }
    case LOAD_TEMPLATE:
      let source = state.itemMap[action.selected].source
      return {
        ...state,
        selected: action.selected,
        source: source
      }
    case SET_SOURCE: {
      return {
        ...state,
        source: action.source
      }
    }
    case SAVE_TEMPLATE: {
      let template = action.template
      if (isError(template)) return state // shouldn't happen, but if it does, do nothing
      if (state.itemMap[template.name] !== undefined) return state // same
      let newItemMap = {
        ...state.itemMap,
      }
      newItemMap[template.name] = template
      return {
        ...state,
        idList: [...state.idList, template.name],
        itemMap: newItemMap
      }
    }
    default:
      return state
  }
}
