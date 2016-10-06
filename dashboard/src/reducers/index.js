import { combineReducers } from 'redux'
import { routerReducer as routing} from 'react-router-redux'
import { reducer as form } from 'redux-form'

import { reducers as access_token } from 'features/accessTokens'
import { reducers as account } from 'features/accounts'
import { reducers as app } from 'features/app'
import { reducers as asset } from 'features/assets'
import { reducers as configuration } from 'features/configuration'
import { reducers as core } from 'features/core'
import { reducers as transaction } from 'features/transactions'
import balance from './balance'
import mockhsm from './mockhsm'
import unspent from './unspent'

const makeRootReducer = () => (state, action) => {
  if (action.type == 'UPDATE_CORE_INFO' &&
      !action.param.is_configured) {
    const newState = {
      form: state.form,
      routing: state.routing
    }

    if (state.core.blockchainID == (action.param.blockchain_id || 0)) {
      newState.core = state.core
    }

    state = newState
  }

  return combineReducers({
    ...access_token,
    account,
    app,
    asset,
    balance,
    configuration,
    core,
    form,
    mockhsm,
    routing,
    transaction,
    unspent,
  })(state, action)
}
export default makeRootReducer
