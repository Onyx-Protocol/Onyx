// external imports
import { routerReducer } from 'react-router-redux'
import { combineReducers } from 'redux'

// ivy imports
import accounts from '../accounts'
import assets from '../assets'
import contracts from '../contracts'
import templates from '../templates'

// internal imports
import * as actions from './actions'
import * as types from './types'

export default function reducer(state: types.AppState, action): types.AppState {
  switch (action.type) {
    case actions.RESET:
      let contractsState = contracts.reducer(undefined, {})
      let fixedContractState = {
        ...contractsState,
        firstTime: false
      }
      return {
        accounts: accounts.reducer(undefined, {}),
        assets: assets.reducer(undefined, {}),
        templates: templates.reducer(undefined, {}),
        routing: state.routing,
        contracts: fixedContractState
      }
    default:
      return combineReducers({
        accounts: accounts.reducer,
        assets: assets.reducer,
        contracts: contracts.reducer,
        templates: templates.reducer,
        routing: routerReducer
      })(state, action) as types.AppState
  }
}
