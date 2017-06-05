// ivy imports
import { SEED } from '../app/actions'

// internal imports
import { FETCH, INITIAL_STATE } from './constants'
import { Item, ItemMap, State } from './types'

export default function reducer(state: State = INITIAL_STATE, action): State {
  switch(action.type) {
    case FETCH: {
      const itemMap = action.items.reduce((map: ItemMap, acct: Item) => {
        const id: string = acct.id
        const alias: string = acct.alias
        return { ...map, [id]: { id, alias } }
      }, {})

      // Sort accounts in alphabetical order by alias
      const idList = [...action.items].sort((a,b) => {
        if (a.alias < b.alias) {
          return -1
        }
        if (a.alias > b.alias) {
          return 1
        }
        return 0
      }).map(item => item.id)

      // {
      //   [acctId: string]:  {
      //     [assetId: string]: amount: number
      //   }
      // }
      const balanceMap = action.items.reduce((map, acct: Item, i: number) => {
        const balances = action.balances[i].items.reduce((map, item) => {
          return {
            ...map,
            [item.sumBy.assetId]: item.amount
          }
        }, {})
        return {
          ...map,
          [acct.id]: balances
        }
      }, {})
      return {
        itemMap,
        idList,
        balanceMap,
        shouldSeed: false
      }
    }
    case SEED: {
      return {
        ...state,
        shouldSeed: false
      }
    }
    default: {
      return {
        ...state,
        shouldSeed: false
      }
    }
  }
}
