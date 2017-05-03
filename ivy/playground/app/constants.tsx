import { routerReducer } from 'react-router-redux'

import accounts from '../accounts'
import assets from '../assets'
import contracts from '../contracts'
import templates from '../templates'

export const NAME: string = "app"
export const RESET: string = "app/RESET"
export const INITIAL_STATE = {
  accounts: accounts.reducer(undefined, {}),
  assets: assets.reducer(undefined, {}),
  contracts: contracts.reducer(undefined, {}),
  templates: templates.reducer(undefined, {}),
  routing: routerReducer(undefined, {})
}

