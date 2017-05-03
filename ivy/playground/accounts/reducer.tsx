import { FETCH, INITIAL_STATE } from './constants'
import { Item, ItemMap, State } from './types'

export default function reducer(state: State = INITIAL_STATE, action): State {
  switch(action.type) {
    case FETCH: {
      const itemMap = action.items.reduce((map: ItemMap, item: Item) => {
        const id: string = item.id
        const alias: string = item.alias
        return { ...map, [id]: { id, alias } }
      }, {})
      const idList = action.items.map(item => item.id)
      return { itemMap, idList }
    }
    default: return state
  }
}
