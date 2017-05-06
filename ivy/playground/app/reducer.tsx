import { routerReducer } from 'react-router-redux'
import { combineReducers } from 'redux'

import accounts from '../accounts'
import assets from '../assets'
import contracts from '../contracts'
import templates from '../templates'

import { AppState } from './types'
import { RESET, INITIAL_STATE } from './constants'

export default function reducer(state: AppState, action): AppState {
  switch (action.type) {
    case RESET:
      return {
        ...INITIAL_STATE,
        templates: state.templates
      }
    default:
      return combineReducers({
        accounts: accounts.reducer,
        assets: assets.reducer,
        contracts: contracts.reducer,
        templates: templates.reducer,
        routing: routerReducer
      })(state, action) as AppState
  }
}
