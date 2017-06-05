import { State } from './types'

export const FETCH: string = "accounts/FETCH"
export const NAME: string = "accounts"
export const INITIAL_STATE: State = {
  itemMap: {},
  idList: [],
  balanceMap: {},
  shouldSeed: true
}

