import { getItemMap, getIdList, getTemplate } from './selectors'

export const LOAD_TEMPLATE = 'templates/LOAD_TEMPLATE'
export const SET_SOURCE = 'templates/SET_SOURCE'

export const load = (selected: string) => {
  return {
    type: LOAD_TEMPLATE,
    selected: selected
  }
}

export const setSource = (source: string) => {
  return (dispatch, getState) => {
    const type = SET_SOURCE
    return dispatch({ type, source })
  }
}

export const SAVE_TEMPLATE = 'SAVE_TEMPLATE'

export function saveTemplate() {
  return (dispatch, getState) => {
    let template = getTemplate(getState())
    dispatch({
      type: SAVE_TEMPLATE,
      template: template
    })
  }
}
