import { createSelector } from 'reselect'

import * as app from '../app/types'
import { State, Item, ItemMap, CompilerError } from './types'
import { compileTemplate } from 'ivy-compiler'

export const getState = (state: app.AppState): State => state.templates

export const getSource = createSelector(
  getState,
  (state: State): string => state.source
)

export const getSelectedTemplateId = createSelector(
  getState,
  (state: State): string => state.selected
)

export const getItemMap = createSelector(
  getState,
  (state: State): ItemMap => state.itemMap
)

export const getSelectedTemplate = createSelector(
  getItemMap,
  getSelectedTemplateId,
  (itemMap, id): Item => itemMap[id]
)

export const getIdList = createSelector(
  getState,
  (state: State): string[] => state.idList
)

export const getItem = (id: string) => {
  return createSelector(
    getItemMap,
    (itemMap: ItemMap): Item | undefined => {
      return itemMap[id]
    }
  )
}

export const getCompiled = createSelector(
  getState,
  (state) => state.compiled
)

export const getOpcodes = createSelector(
  getCompiled,
  (compiled) => compiled && compiled.opcodes
)

export const getTemplate = createSelector(
  getSource,
  (source: string): Item | CompilerError => {
    return compileTemplate(source)
  }
)

export const getParameterIdList = createSelector(
  getTemplate,
  (template: Item): string[] => {
    return template.contractParameters
      .map(param => "contractParameters." + param.identifier)
  }
)

export const getDataParameterIdList = createSelector(
  getTemplate,
  (template: Item): string[] => {
    return template.contractParameters
      .filter(param => param.valueType !== "Value" )
      .map(param => "contractParameters." + param.identifier)
  }
)

