import { combineReducers } from 'redux'
import { routerReducer as routing} from 'react-router-redux'
import { reducer as form } from 'redux-form'

import app from './app'
import core from './core'
import transaction from './transaction'
import unspent from './unspent'
import balance from './balance'
import account from './account'
import asset from './asset'
import index from './indexQuery'
import mockhsm from './mockhsm'

export default combineReducers({
  app,
  routing,
  form,
  core,
  transaction,
  unspent,
  balance,
  asset,
  account,
  index,
  mockhsm
})
