import { combineReducers } from 'redux'
import { routerReducer as routing} from 'react-router-redux'
import { reducer as form } from 'redux-form'


import { reducers as account } from '../features/accounts'
import { reducers as asset } from '../features/assets'
import { reducers as transaction } from '../features/transactions'
import app from './app'
import balance from './balance'
import core from './core'
import mockhsm from './mockhsm'
import unspent from './unspent'

export default combineReducers({
  account,
  app,
  asset,
  balance,
  core,
  form,
  mockhsm,
  routing,
  transaction,
  unspent,
})
