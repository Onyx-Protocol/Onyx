// ivy imports
import { SEED } from '../app/actions'

// internal imports
import { Item, ItemMap, State } from './types'
import { FETCH, INITIAL_STATE } from './constants'

export default function reducer(state: State = INITIAL_STATE, action): State {
  switch(action.type) {
    case FETCH: {
      const itemMap = action.items.reduce((map: ItemMap, item: Item) => {
        const id: string = item.id
        const alias: string = item.alias
        return { ...map, [id]: { id, alias } }
      }, {})

      // Sort assets in alphabetical order
      const idList = [...action.items].sort((a,b) => {
        if (a.alias < b.alias) {
          return -1
        }
        if (a.alias > b.alias) {
          return 1
        }
        return 0
      }).map(item => item.id)
      return {
        itemMap,
        idList,
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
