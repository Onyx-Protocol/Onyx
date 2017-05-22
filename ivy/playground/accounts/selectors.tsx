import * as React from 'react'
import { createSelector } from 'reselect'

import * as app from '../app/types'
import { getState as getContractsState } from '../contracts/selectors'
import { getInputMap } from '../templates/selectors'

import { State, Item, ItemMap } from './types'

const getState = (state: app.AppState): State => state.accounts

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

export const getBalanceMap = createSelector(
  getState,
  (state: State) => state.balanceMap
)

export const getBalanceSelector = (namePrefix: string) => {
  return createSelector(
    getBalanceMap,
    getInputMap,
    getContractsState,
    (balanceMap, inputMap, contracts) => {
      if (inputMap === undefined) {
        return undefined
      }

      let acctInput
      let assetInput
      if (namePrefix.startsWith("contract")) {
        acctInput = inputMap[namePrefix + ".accountInput"]
        assetInput = inputMap[namePrefix + ".assetInput"]
      } else if (namePrefix.startsWith("clause")) {
        // THIS IS A HACK
        const spendInputMap = contracts.contractMap[contracts.spendContractId].spendInputMap
        acctInput = spendInputMap[namePrefix + ".valueInput.accountInput"]
        assetInput = spendInputMap[namePrefix + ".valueInput.assetInput"]
      }

      let balance
      if (acctInput && acctInput.value && assetInput && assetInput.value) {
        balance = balanceMap[acctInput.value][assetInput.value]
        if (balance === undefined) {
          balance = 0
        }
      }
      return balance
    }
  )
}

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
