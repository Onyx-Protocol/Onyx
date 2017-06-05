import { createSelector } from 'reselect'

import * as app from '../app/types'
import { State, Item, ItemMap } from './types'

const getState = (state: app.AppState): State => state.assets

export const getItemMap = createSelector(
  getState,
  (state: State): ItemMap => state.itemMap
)

export const getIdList = createSelector(
  getState,
  (state: State): string[]  => state.idList
)

export const getItemList = createSelector(
  getItemMap,
  getIdList,
  (itemMap: ItemMap, list: string[]): Item[] => {
    return list.map(id => itemMap[id])
  }
)

export const getShouldSeed = createSelector(
  getState,
  (state: State) => state.shouldSeed
)

export const getItem = (id) => {
  return createSelector(
    getState,
    getItemMap,
    (state: State, itemMap: ItemMap): Item | undefined => {
      return itemMap[id]
    }
  )
}
